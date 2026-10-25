package logger

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/noop"
)

type recordProvider struct {
	embedded.LoggerProvider
	recs []log.Record
}

func (p *recordProvider) Logger(string, ...log.LoggerOption) log.Logger {
	return &recordLogger{p: p}
}

type recordLogger struct {
	embedded.Logger
	p *recordProvider
}

func (l *recordLogger) Emit(_ context.Context, r log.Record) {
	l.p.recs = append(l.p.recs, r.Clone())
}

func (l *recordLogger) Enabled(context.Context, log.EnabledParameters) bool { return true }

// Verifies that the hook forwards severity and body to the global logger provider.
func TestOtelHook(t *testing.T) {
	provider := &recordProvider{}
	global.SetLoggerProvider(provider)
	t.Cleanup(func() { global.SetLoggerProvider(noop.NewLoggerProvider()) })

	h := newOtelHook("svc")
	zl := zerolog.New(nil)
	h.Run(zl.Warn(), zerolog.WarnLevel, "hello")

	if len(provider.recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(provider.recs))
	}

	rec := provider.recs[0]
	if got := rec.Body().AsString(); got != "hello" {
		t.Fatalf("body = %q, want %q", got, "hello")
	}
	if got := rec.Severity(); got != log.SeverityWarn {
		t.Fatalf("severity = %v, want %v", got, log.SeverityWarn)
	}
}
