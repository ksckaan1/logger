package logger

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	zlog         *zerolog.Logger
	serviceNames []string
	cfg          *Config
	file         io.WriteCloser
	otel         *otelEmitter
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
		cfg:  cfg,
		otel: newOtelEmitter(cfg.ServiceName),
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
		otel:         l.otel,
	}
}

func (l *Logger) Trace(ctx context.Context, message string, fields ...any) {
	l.log(ctx, zerolog.TraceLevel, l.zlog.Trace(), message, fields...)
}

func (l *Logger) Debug(ctx context.Context, message string, fields ...any) {
	l.log(ctx, zerolog.DebugLevel, l.zlog.Debug(), message, fields...)
}

func (l *Logger) Info(ctx context.Context, message string, fields ...any) {
	l.log(ctx, zerolog.InfoLevel, l.zlog.Info(), message, fields...)
}

func (l *Logger) Warn(ctx context.Context, message string, fields ...any) {
	l.log(ctx, zerolog.WarnLevel, l.zlog.Warn(), message, fields...)
}

func (l *Logger) Error(ctx context.Context, message string, fields ...any) {
	l.log(ctx, zerolog.ErrorLevel, l.zlog.Error(), message, fields...)
}

func (l *Logger) Fatal(ctx context.Context, message string, fields ...any) {
	l.log(ctx, zerolog.FatalLevel, l.zlog.Fatal(), message, fields...)
}

func (l *Logger) Panic(ctx context.Context, message string, fields ...any) {
	l.log(ctx, zerolog.PanicLevel, l.zlog.Panic(), message, fields...)
}

func (l *Logger) log(ctx context.Context, level zerolog.Level, e *zerolog.Event, message string, fields ...any) {
	if !e.Enabled() {
		return
	}

	var attrs []attribute.KeyValue

	if len(l.serviceNames) != 0 {
		service := strings.Join(l.serviceNames, "/")
		e.Str("service", service)
		attrs = append(attrs, attribute.String("service", service))
	}

	if l.cfg.InjectTraceInfo {
		e = l.addTraceAndSpanID(ctx, e)
		attrs = append(attrs, traceAttrs(ctx)...)
	}

	attrs = append(attrs, otelAttrs(fields)...)
	if caller, ok := callerAttr(); ok {
		attrs = append(attrs, caller)
	}

	// Emit before Msg, because Fatal and Panic terminate the process in Msg.
	l.otel.emit(ctx, level, message, attrs)

	e.Fields(fields).Msg(message)
}

// callerAttr mirrors the caller field zerolog adds, as an OTel attribute.
//
// The stack is walked instead of using a fixed skip count, because the small
// level methods get inlined into the caller, which would shift a fixed count.
func callerAttr() (attribute.KeyValue, bool) {
	pc, file, line, ok := callerFrame()
	if !ok {
		return attribute.KeyValue{}, false
	}
	return attribute.String(zerolog.CallerFieldName, zerolog.CallerMarshalFunc(pc, file, line)), true
}

// loggerPkg is the runtime prefix of this package, e.g.
// "github.com/ksckaan1/logger." including the trailing dot.
var loggerPkg = reflect.TypeOf(Logger{}).PkgPath() + "."

// isLoggerFrame reports whether name belongs to this logger's own call path.
// Only the Logger methods and helpers are skipped, so that test or caller
// functions living in the same package are still reported.
func isLoggerFrame(name string) bool {
	if !strings.HasPrefix(name, loggerPkg) {
		return false
	}
	return strings.Contains(name, "(*Logger).") ||
		strings.HasSuffix(name, ".callerAttr") ||
		strings.HasSuffix(name, ".callerFrame")
}

func callerFrame() (uintptr, string, int, bool) {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(0, pcs)

	frames := runtime.CallersFrames(pcs[:n])
	for {
		f, more := frames.Next()

		// Skip the runtime and this logger; the first other frame is the code
		// that called the logger.
		if !strings.HasPrefix(f.Function, "runtime.") && !isLoggerFrame(f.Function) {
			return f.PC, f.File, f.Line, true
		}

		if !more {
			return 0, "", 0, false
		}
	}
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
	l.file, err = os.OpenFile(l.cfg.OutputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	return nil
}

func traceAttrs(ctx context.Context) []attribute.KeyValue {
	spanCtx := trace.SpanContextFromContext(ctx)

	var attrs []attribute.KeyValue
	if spanCtx.HasSpanID() {
		attrs = append(attrs, attribute.String("span_id", spanCtx.SpanID().String()))
	}
	if spanCtx.HasTraceID() {
		attrs = append(attrs, attribute.String("trace_id", spanCtx.TraceID().String()))
	}
	return attrs
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
