package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubectl/pkg/drain"
)

const (
	metadataURI        = "http://169.254.169.254/latest/meta-data/spot/instance-action"
	podNameEnv         = "POD_NAME"
	nodeNameEnv        = "NODE_NAME"
	forceEnv           = "FORCE"
	deleteEmptyDirEnv  = "DELETE_EMPTY_DIR"
	ignoreDaemonSetEnv = "IGNORE_DAEMONSETS"
	gracePeriodEnv     = "GRACE_PERIOD"
	devModeEnv         = "DEV_MODE"
	logLevelEnv        = "LOG_LEVEL"
)

var (
	devMode             = false
	force               = true
	gracePeriodSeconds  = 120
	ignoreAllDaemonSets = true
	deleteEmptyDirData  = true
	clientSet           *kubernetes.Clientset
	podName             string
	nodeName            string
)

func init() {
	podName = os.Getenv(podNameEnv)
	if podName == "" {
		panic(fmt.Sprintf("environment variable %s for current not set or empty", podNameEnv))
	}

	nodeName = os.Getenv(nodeNameEnv)
	if nodeName == "" {
		panic(fmt.Sprintf("environment variable %s not set or empty. It's is a node that will be drained", nodeNameEnv))
	}

	if value, err := strconv.ParseBool(os.Getenv(devModeEnv)); err == nil {
		devMode = value
	}

	if value, err := strconv.ParseBool(os.Getenv(forceEnv)); err == nil {
		force = value
	}

	if value, err := strconv.ParseBool(os.Getenv(deleteEmptyDirEnv)); err == nil {
		deleteEmptyDirData = value
	}

	if value, err := strconv.ParseBool(os.Getenv(ignoreDaemonSetEnv)); err == nil {
		ignoreAllDaemonSets = value
	}

	if value, err := strconv.Atoi(os.Getenv(gracePeriodEnv)); err == nil {
		gracePeriodSeconds = value
	}
}

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	ctx := context.Background()

	logger := buildLogger()
	defer func() {
		_ = logger.Sync()
	}()

	log := logger.Named("main").Sugar()

	config, err := getKubeConfig(log)
	if err != nil {
		log.Panic(err)
	}

	clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err)
	}

	log.Info(fmt.Sprintf(
		"starting spot-termination-handler: %s=%t %s=%t %s=%s %s=%s %s=%t %s=%t %s=%d",
		devModeEnv, devMode,
		forceEnv, force,
		nodeNameEnv, nodeName,
		podNameEnv, podName,
		ignoreDaemonSetEnv, ignoreAllDaemonSets,
		deleteEmptyDirEnv, deleteEmptyDirData,
		gracePeriodEnv, gracePeriodSeconds,
	))

	dh := &drain.Helper{
		Ctx:                 ctx,
		Client:              clientSet,
		Force:               force,
		GracePeriodSeconds:  gracePeriodSeconds,
		IgnoreAllDaemonSets: ignoreAllDaemonSets,
		DeleteEmptyDirData:  deleteEmptyDirData,
		AdditionalFilters: []drain.PodFilter{
			func(pod v1.Pod) drain.PodDeleteStatus {
				if pod.Name == podName {
					return drain.MakePodDeleteStatusSkip()
				}
				return drain.MakePodDeleteStatusOkay()
			},
		},
		Out: &drainWriter{
			level: zapcore.InfoLevel,
			log:   logger.Named("drain"),
		},
		ErrOut: &drainWriter{
			level: zapcore.ErrorLevel,
			log:   logger.Named("drain"),
		},
		OnPodDeletedOrEvicted: func(pod *v1.Pod, usingEviction bool) {
			if er := generateSpotInterruptionEvent(ctx, pod); er != nil {
				log.Errorf("failed to generate event for pod %s: %s", pod.Name, er)
			}
			log.Infof("%s in namespace %s, evicted", pod.Name, pod.Namespace)
		},
	}

	node, err := clientSet.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for {
		resp, err := http.Get(metadataURI)
		if err != nil {
			log.Warnf("the HTTP request failed with error %s\n", err)
			continue
		}

		_, _ = io.Copy(ioutil.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode == 200 {
			log.Info("draining node - spot node is being terminated")
			if err := drain.RunCordonOrUncordon(dh, node, true); err != nil {
				log.Errorf("unable to cordon node %s", err)
			} else {
				if pods, err := dh.GetPodsForDeletion(node.Name); err != nil {
					log.Errorf("unable to list pods %s\n", err)
				} else {
					if e := dh.DeleteOrEvictPods(pods.Pods()); e != nil {
						log.Errorf("failed to evict pods %s", e)
					}
				}
			}
			break
		}
		time.Sleep(5 * time.Second)
	}
}

func getKubeConfig(log *zap.SugaredLogger) (*rest.Config, error) {
	if devMode {
		log.Debugf("using kubeconfig from homedir %s is set", devModeEnv)
		// use the current context in kubeconfig
		return clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	}
	log.Debug("using incluster config")
	return rest.InClusterConfig()
}

func buildLogger() *zap.Logger {
	var logLevel string
	if logLevel = os.Getenv(logLevelEnv); logLevel == "" {
		logLevel = "DEBUG"
	}

	logCfg := zap.NewProductionConfig()
	if devMode {
		logCfg = zap.NewDevelopmentConfig()
	}

	logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var level zapcore.Level
	if err := level.Set(logLevel); err != nil {
		panic(err)
	}
	logCfg.Level.SetLevel(level)
	logger, err := logCfg.Build()
	if err != nil {
		panic(err)
	}

	return logger
}
