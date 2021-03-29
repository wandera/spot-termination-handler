package main

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type drainWriter struct {
	level zapcore.Level
	log   *zap.Logger
}

func (d *drainWriter) Write(p []byte) (n int, err error) {
	d.log.Check(d.level, strings.TrimSpace(string(p))).Write()
	return
}
