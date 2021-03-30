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

func (d *drainWriter) Write(p []byte) (int, error) {
	if entry := d.log.Check(d.level, strings.TrimSpace(string(p))); entry != nil {
		entry.Write()
		return len(p), nil
	}
	return 0, nil
}
