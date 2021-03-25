package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	MetadataURI = "http://169.254.169.254/latest/meta-data/spot/instance-action"
)

func main() {
	logger := buildLogger()
	defer func() {
		_ = logger.Sync()
	}()

	log := zap.S().Named("main")

	drainParams := os.Getenv("DRAIN_PARAMETERS")
	if drainParams == "" {
		drainParams = "--grace-period=120 --force --ignore-daemonsets --delete-local-data"
	}

	log.Infof("Kubectl drain parameters: %s", drainParams)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	log.Info("Starting spot-termination-handler")

	for {
		if resp, err := http.Get(MetadataURI); err != nil {
			log.Warnf("The HTTP request failed with error %s\n", err)
		} else if resp.Status == "200" {
			log.Info("Draining node - spot node is being terminated.")
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func buildLogger() *zap.Logger {
	var logLevel string
	if logLevel = os.Getenv("LOG_LEVEL"); logLevel == "" {
		logLevel = "DEBUG"
	}
	var devMode string
	if devMode = os.Getenv("DEV_MODE"); devMode == "" {
		devMode = "1"
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
