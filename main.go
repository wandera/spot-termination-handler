package main

import (
	"context"
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubectl/pkg/drain"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	MetadataURI         = "http://169.254.169.254/latest/meta-data/spot/instance-action"
	force               = true
	gracePeriodSeconds  = 120
	ignoreAllDaemonSets = true
	deleteEmptyDirData  = true
)

func main(){
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	ctx := context.Background()

	var devMode string
	if devMode = os.Getenv("DEV_MODE"); devMode == "" {
		devMode = "1"
	}

	logger := buildLogger(devMode)
	defer func() {
		_ = logger.Sync()
	}()

	log := zap.S().Named("main")

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Panic("Environment variable NODE_NAME not set or empty. It's is a node that should be drained!")
	}

	config, err := getKubeConfig(log, devMode)
	if err != nil {
		log.Panic(err)
	}

	log.Info("Starting spot-termination-handler")
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err)
	}

	dh := drain.Helper{
		Ctx:                             ctx,
		Client:                          clientset,
		Force:                           force,
		GracePeriodSeconds:              gracePeriodSeconds,
		IgnoreAllDaemonSets:             ignoreAllDaemonSets,
		DeleteEmptyDirData:              deleteEmptyDirData,
		Out:                             nil,
		ErrOut:                          nil,
		OnPodDeletedOrEvicted: func(pod *v1.Pod, usingEviction bool) {
			log.Infof("%s in namespace %s, evicted", pod.Name, pod.Namespace)
		},
	}

	for {
		if resp, err := http.Get(MetadataURI); err != nil {
			log.Warnf("The HTTP request failed with error %s\n", err)
		} else if resp.Status == "200" {
			log.Info("Draining node - spot node is being terminated.")
			if pods, err := dh.GetPodsForDeletion(nodeName); err != nil {
				log.Errorf("Unable to list pods %s\n", err)
			} else {
				if e := dh.DeleteOrEvictPods(pods.Pods()); e != nil{
					log.Errorf("Failed to evict pods %s", e)
				}
			}
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func getKubeConfig(log *zap.SugaredLogger, devMode string) (*rest.Config, error) {
	var config *rest.Config

	if devMode == "1" {
		log.Debug("DEV_MODE is set. Using kubconfig from homedir.")
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
		log.Debug("DEV_MODE is not set. Using incluster config.")
		var err error
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func buildLogger(devMode string) *zap.Logger {
	var logLevel string
	if logLevel = os.Getenv("LOG_LEVEL"); logLevel == "" {
		logLevel = "DEBUG"
	}

	logCfg := zap.NewProductionConfig()
	if devMode == "1" {
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
		zap.S().Panicf("failed to build logger: %v", err)
	}
	zap.ReplaceGlobals(logger)
	return logger
}
