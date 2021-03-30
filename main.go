package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"spot-termination-handler/pkg/logs"
	"spot-termination-handler/pkg/terminate"
	"strconv"
	"syscall"

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
	logLevel            = "DEBUG"
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

	if value := os.Getenv(logLevelEnv); value != "" {
		logLevel = value
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

	// load the node early - cache or fail-fast if unavailable
	node, err := clientSet.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
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
		AdditionalFilters:   []drain.PodFilter{filterPod(podName)},
		Out:                 logs.NewZapWriter(zapcore.InfoLevel, logger.Named("drain")),
		ErrOut:              logs.NewZapWriter(zapcore.ErrorLevel, logger.Named("drain")),
		OnPodDeletedOrEvicted: func(pod *v1.Pod, usingEviction bool) {
			if er := generateSpotInterruptionEvent(ctx, pod); er != nil {
				log.Errorf("failed to generate event for pod %s: %s", pod.Name, er)
			}
			log.Infof("%s in namespace %s, evicted", pod.Name, pod.Namespace)
		},
	}

	select {
	case <-terminate.WaitCh():
		log.Info("draining node - spot node is being terminated")
		if err := drain.RunCordonOrUncordon(dh, node, true); err != nil {
			log.Errorf("unable to cordon node %s", err)
		} else {
			if pods, err := dh.GetPodsForDeletion(node.Name); err != nil {
				log.Errorf("unable to list pods %s\n", err)
			} else {
				if err := dh.DeleteOrEvictPods(pods.Pods()); err != nil {
					log.Errorf("failed to evict pods %s", err)
				}
			}
		}
	case <-shutdown:
	}
	log.Info("shutting down spot-termination-handler")
}

func filterPod(podName string) func(pod v1.Pod) drain.PodDeleteStatus {
	return func(pod v1.Pod) drain.PodDeleteStatus {
		if pod.Name == podName {
			return drain.MakePodDeleteStatusSkip()
		}
		return drain.MakePodDeleteStatusOkay()
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
