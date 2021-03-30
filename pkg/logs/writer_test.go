package logs

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func Test_drainWriter_Write(t *testing.T) {
	type fields struct {
		level zapcore.Level
		log   *zap.Logger
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "error logs",
			fields: fields{
				level: zapcore.ErrorLevel,
				log:   zaptest.NewLogger(t),
			},
			args: args{
				p: []byte("foo"),
			},
			want: 3,
		},
		{
			name: "info logs",
			fields: fields{
				level: zapcore.InfoLevel,
				log:   zaptest.NewLogger(t),
			},
			args: args{
				p: []byte("foo"),
			},
			want: 3,
		},
		{
			name: "no print error logs",
			fields: fields{
				level: zapcore.ErrorLevel,
				log:   zaptest.NewLogger(t, zaptest.Level(zap.PanicLevel)),
			},
			args: args{
				p: []byte("foo"),
			},
			want: 0,
		},
		{
			name: "no print info logs",
			fields: fields{
				level: zapcore.InfoLevel,
				log:   zaptest.NewLogger(t, zaptest.Level(zap.PanicLevel)),
			},
			args: args{
				p: []byte("foo"),
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &zapWriter{
				level: tt.fields.level,
				log:   tt.fields.log,
			}
			got, err := d.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Write() got = %v, want %v", got, tt.want)
			}
		})
	}
}
