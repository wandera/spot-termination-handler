package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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
	metadataURI = "http://169.254.169.254/latest/meta-data/spot/instance-action"
)

var (
	force               = true
	gracePeriodSeconds  = 120
	ignoreAllDaemonSets = true
	deleteEmptyDirData  = true
	clientSet           *kubernetes.Clientset
	reportingInstance   string
	nodeName            string
)

func init() {
	reportingInstance = os.Getenv("POD_NAME")
	if reportingInstance == "" {
		panic("environment variable POD_NAME for current not set or empty")
	}

	nodeName = os.Getenv("NODE_NAME")
	if nodeName == "" {
		panic("environment variable NODE_NAME not set or empty. It's is a node that will be drained")
	}

	if value, err := strconv.ParseBool(os.Getenv("FORCE")); err == nil {
		force = value
	}

	if value, err := strconv.ParseBool(os.Getenv("DELETE_EMPTY_DIR")); err == nil {
		deleteEmptyDirData = value
	}

	if value, err := strconv.ParseBool(os.Getenv("IGNORE_DAEMONSETS")); err == nil {
		ignoreAllDaemonSets = value
	}

	if value, err := strconv.Atoi(os.Getenv("GRACE_PERIOD")); err == nil {
		gracePeriodSeconds = value
	}
}

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	ctx := context.Background()

	devMode := os.Getenv("DEV_MODE") == "1"

	logger := buildLogger(devMode)
	defer func() {
		_ = logger.Sync()
	}()

	log := logger.Named("main").Sugar()

	config, err := getKubeConfig(log, devMode)
	if err != nil {
		log.Panic(err)
	}

	log.Info(fmt.Sprintf(
		"starting spot-termination-handler: DEV_MODE=%t FORCE=%t POD_NAME=%s IGNORE_DAEMONSETS=%t "+
			"DELETE_EMPTY_DIR=%t GRACE_PERIOD=%d ", devMode, force, reportingInstance, ignoreAllDaemonSets,
		deleteEmptyDirData, gracePeriodSeconds))
	clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err)
	}

	dh := &drain.Helper{
		Ctx:                 ctx,
		Client:              clientSet,
		Force:               force,
		GracePeriodSeconds:  gracePeriodSeconds,
		IgnoreAllDaemonSets: ignoreAllDaemonSets,
		DeleteEmptyDirData:  deleteEmptyDirData,
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

		_, _ = io.Copy(io.Discard, resp.Body)
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

func getKubeConfig(log *zap.SugaredLogger, devMode bool) (*rest.Config, error) {
	var config *rest.Config

	if devMode {
		log.Debug("using kubeconfig from homedir DEV_MODE is set")
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		// use the current context in kubeconfig
		var err error
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		log.Debug("using incluster config DEV_MODE is not set")
		var err error
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func buildLogger(devMode bool) *zap.Logger {
	var logLevel string
	if logLevel = os.Getenv("LOG_LEVEL"); logLevel == "" {
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
