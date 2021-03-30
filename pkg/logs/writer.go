package logs

import (
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapWriter(level zapcore.Level, log *zap.Logger) io.Writer {
	return &zapWriter{
		level: level,
		log:   log,
	}
}

type zapWriter struct {
	level zapcore.Level
	log   *zap.Logger
}

func (d *zapWriter) Write(p []byte) (int, error) {
	if entry := d.log.Check(d.level, strings.TrimSpace(string(p))); entry != nil {
		entry.Write()
		return len(p), nil
	}
	return 0, nil
}
