package logger

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	zlog         *zerolog.Logger
	serviceNames []string
	cfg          *Config
	file         io.WriteCloser
}

func New(cfg *Config) (*Logger, error) {
	if cfg.parseEnv {
		err := parseEnv(cfg)
		if err != nil {
			return nil, fmt.Errorf("parseEnv: %w", err)
		}
		cfg.parseEnv = true
	}
	cfg.setDefaults()

	vld := validator.New()

	err := vld.Struct(cfg)
	if err != nil {
		return nil, fmt.Errorf("vld.Struct: %w", err)
	}

	lg := &Logger{
		cfg: cfg,
	}
	if cfg.ServiceName != "" {
		lg.serviceNames = append(lg.serviceNames, cfg.ServiceName)
	}
	zlog, err := lg.initZerolog()
	if err != nil {
		return nil, fmt.Errorf("initZerolog: %w", err)
	}
	lg.zlog = zlog

	return lg, nil
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Sub(serviceName string) *Logger {
	serviceNames := l.serviceNames
	if serviceName != "" {
		serviceNames = append(serviceNames, serviceName)
	}
	return &Logger{
		zlog:         l.zlog,
		serviceNames: serviceNames,
		cfg:          l.cfg,
	}
}

func (l *Logger) Trace(ctx context.Context, message string, fields ...any) {
	l.log(ctx, l.zlog.Trace(), message, fields...)
}

func (l *Logger) Debug(ctx context.Context, message string, fields ...any) {
	l.log(ctx, l.zlog.Debug(), message, fields...)
}

func (l *Logger) Info(ctx context.Context, message string, fields ...any) {
	l.log(ctx, l.zlog.Info(), message, fields...)
}

func (l *Logger) Warn(ctx context.Context, message string, fields ...any) {
	l.log(ctx, l.zlog.Warn(), message, fields...)
}

func (l *Logger) Error(ctx context.Context, message string, fields ...any) {
	l.log(ctx, l.zlog.Error(), message, fields...)
}

func (l *Logger) Fatal(ctx context.Context, message string, fields ...any) {
	l.log(ctx, l.zlog.Fatal(), message, fields...)
}

func (l *Logger) Panic(ctx context.Context, message string, fields ...any) {
	l.log(ctx, l.zlog.Panic(), message, fields...)
}

func (l *Logger) log(ctx context.Context, e *zerolog.Event, message string, fields ...any) {
	if len(l.serviceNames) != 0 {
		e.Str("service", strings.Join(l.serviceNames, "/"))
	}
	if l.cfg.InjectTraceInfo {
		e = l.addTraceAndSpanID(ctx, e)
	}
	e.Fields(fields).Msg(message)
}

func (l *Logger) initZerolog() (*zerolog.Logger, error) {
	writers, err := l.initWriters()
	if err != nil {
		return nil, fmt.Errorf("initWriters: %w", err)
	}
	mw := io.MultiWriter(writers...)
	zlog := zerolog.
		New(mw).
		Level(l.getZerologLevel(l.cfg.Level)).
		With().
		Timestamp().
		CallerWithSkipFrameCount(4).
		Logger()
	return &zlog, nil
}

func (l *Logger) format(format string, wr io.Writer) io.Writer {
	switch format {
	case "logfmt":
		wr = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = wr
			w.TimeFormat = time.RFC3339
		})
	case "logfmt_no_color":
		wr = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = wr
			w.NoColor = true
			w.TimeFormat = time.RFC3339
		})
	}
	return wr
}

func (l *Logger) initWriters() ([]io.Writer, error) {
	writers := []io.Writer{}
	for _, op := range l.cfg.Output {
		var wr io.Writer
		switch op {
		case "stdout":
			format := cmp.Or(l.cfg.StdoutFormat, l.cfg.OutputFormat)
			if format == "" {
				return nil, fmt.Errorf("stdout and output format is empty")
			}
			wr = l.format(format, os.Stdout)
		case "stderr":
			format := cmp.Or(l.cfg.StderrFormat, l.cfg.OutputFormat)
			if format == "" {
				return nil, fmt.Errorf("stderr and output format is empty")
			}
			wr = l.format(format, os.Stderr)
		case "file":
			format := cmp.Or(l.cfg.FileFormat, l.cfg.OutputFormat)
			if format == "" {
				return nil, fmt.Errorf("file and output format is empty")
			}
			err := l.initFile()
			if err != nil {
				return nil, fmt.Errorf("initFile: %w", err)
			}
			wr = l.format(format, l.file)
		}
		writers = append(writers, wr)
	}
	return writers, nil
}

func (l *Logger) getZerologLevel(level string) zerolog.Level {
	switch level {
	case "disabled":
		return zerolog.Disabled
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

func (l *Logger) initFile() error {
	if l.cfg.RotateEnabled {
		l.file = &lumberjack.Logger{
			Filename:   l.cfg.OutputFilePath,
			MaxSize:    l.cfg.RotateMaxSizeMB,
			MaxBackups: l.cfg.RotateMaxBackups,
			MaxAge:     l.cfg.RotateMaxAgeDays,
			Compress:   l.cfg.RotateCompress,
		}
		return nil
	}
	var err error
	l.file, err = os.OpenFile(l.cfg.OutputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	return nil
}

func (l *Logger) addTraceAndSpanID(ctx context.Context, e *zerolog.Event) *zerolog.Event {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		e = e.Str("span_id", spanCtx.SpanID().String())
	}
	if spanCtx.HasTraceID() {
		e = e.Str("trace_id", spanCtx.TraceID().String())
	}
	return e
}
