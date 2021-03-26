package main

import (
	"context"
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	MetadataURI = "http://169.254.169.254/latest/meta-data/spot/instance-action"
)

func main(){
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	var devMode string
	if devMode = os.Getenv("DEV_MODE"); devMode == "" {
		devMode = "1"
	}

	drainParams := os.Getenv("DRAIN_PARAMETERS")
	if drainParams == "" {
		drainParams = "--grace-period=120 --force --ignore-daemonsets --delete-local-data"
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

	log.Infof("Kubectl drain parameters: %s", drainParams)

	config, err := getKubeConfig(log, devMode)
	if err != nil {
		log.Panic(err)
	}

	log.Info("Starting spot-termination-handler")
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err)
	}

	for {
		if resp, err := http.Get(MetadataURI); err != nil {
			log.Warnf("The HTTP request failed with error %s\n", err)
		} else if resp.Status == "200" {
			log.Info("Draining node - spot node is being terminated.")
			pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
				FieldSelector: "spec.nodeName=" + nodeName,
			})
			if err != nil {
				log.Errorf("Unable to list pods %s\n", err)
			}
			for _, pod := range pods.Items {
				log.Infof("Pod %s", pod.Name)
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
