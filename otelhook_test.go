package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/trace"
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

func (l *recordLogger) last(t *testing.T) log.Record {
	t.Helper()
	if len(l.p.recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(l.p.recs))
	}
	return l.p.recs[0]
}

func attrMap(rec log.Record) map[string]attribute.Value {
	attrs := map[string]attribute.Value{}
	rec.WalkAttributes(func(kv attribute.KeyValue) bool {
		attrs[string(kv.Key)] = kv.Value
		return true
	})
	return attrs
}

func newTestLogger(provider *recordProvider, level zerolog.Level) *Logger {
	zl := zerolog.New(io.Discard).Level(level)
	return &Logger{
		zlog:         &zl,
		cfg:          &Config{InjectTraceInfo: true},
		serviceNames: []string{"api", "user"},
		otel:         &otelEmitter{logger: provider.Logger("svc")},
	}
}

// Verifies that every field, the service name and the trace context end up as
// OTel attributes, not just the message and severity.
func TestOtelForwardAllFields(t *testing.T) {
	provider := &recordProvider{}
	lg := newTestLogger(provider, zerolog.DebugLevel)
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	lg.Info(ctx, "hello",
		"user_id", 42,
		"ratio", 0.5,
		"cached", true,
		"tags", []string{"a", "b"},
		"when", time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC),
		"error", errors.New("boom"),
	)

	rec := (&recordLogger{p: provider}).last(t)
	if got := rec.Body().AsString(); got != "hello" {
		t.Fatalf("body = %q, want %q", got, "hello")
	}
	if got := rec.Severity(); got != log.SeverityInfo {
		t.Fatalf("severity = %v, want %v", got, log.SeverityInfo)
	}

	attrs := attrMap(rec)

	if got := attrs["service"].AsString(); got != "api/user" {
		t.Fatalf("service = %q, want %q", got, "api/user")
	}
	if got := attrs["user_id"].AsInt64(); got != 42 {
		t.Fatalf("user_id = %d, want 42", got)
	}
	if got := attrs["ratio"].AsFloat64(); got != 0.5 {
		t.Fatalf("ratio = %v, want 0.5", got)
	}
	if !attrs["cached"].AsBool() {
		t.Fatal("cached attribute is not true")
	}
	if got := attrs["tags"].AsStringSlice(); len(got) != 2 || got[1] != "b" {
		t.Fatalf("tags = %v, want [a b]", got)
	}
	if got := attrs["when"].AsString(); got != "2026-10-25T12:00:00Z" {
		t.Fatalf("when = %q", got)
	}
	if got := attrs["error"].AsString(); got != "boom" {
		t.Fatalf("error = %q, want boom", got)
	}
	if got := attrs["trace_id"].AsString(); got != sc.TraceID().String() {
		t.Fatalf("trace_id = %q, want %q", got, sc.TraceID().String())
	}
	if got := attrs["span_id"].AsString(); got != sc.SpanID().String() {
		t.Fatalf("span_id = %q, want %q", got, sc.SpanID().String())
	}
}

// Verifies that the OTel caller attribute matches the caller field zerolog
// writes, i.e. both point at the code that called the logger.
func TestOtelCallerMatchesZerolog(t *testing.T) {
	var out bytes.Buffer
	provider := &recordProvider{}
	zl := zerolog.New(&out).With().CallerWithSkipFrameCount(4).Logger()
	lg := &Logger{
		zlog: &zl,
		cfg:  &Config{},
		otel: &otelEmitter{logger: provider.Logger("svc")},
	}

	lg.Info(context.Background(), "with-caller")

	var zlogEntry map[string]any
	if err := json.Unmarshal(out.Bytes(), &zlogEntry); err != nil {
		t.Fatalf("zerolog output is not JSON: %v (%q)", err, out.String())
	}
	want, _ := zlogEntry["caller"].(string)
	if want == "" {
		t.Fatalf("zerolog did not write a caller field: %q", out.String())
	}

	rec := (&recordLogger{p: provider}).last(t)
	got := attrMap(rec)["caller"].AsString()
	if got != want {
		t.Fatalf("caller = %q, want %q", got, want)
	}
	if !strings.Contains(want, "otelhook_test.go:") {
		t.Fatalf("caller %q does not point at this test file", want)
	}
}

// Verifies that fields below the configured level are not emitted at all.
func TestOtelRespectsLevel(t *testing.T) {
	provider := &recordProvider{}
	lg := newTestLogger(provider, zerolog.WarnLevel)

	lg.Info(context.Background(), "dropped", "k", "v")

	if len(provider.recs) != 0 {
		t.Fatalf("expected no records, got %d", len(provider.recs))
	}
}

// Verifies that an unpaired key or a non-string key is dropped, like zerolog.
func TestOtelAttrsEdgeCases(t *testing.T) {
	attrs := otelAttrs([]any{5, "dropped", "kept", "v", "unpaired"})

	if len(attrs) != 1 || attrs[0].Key != "kept" {
		t.Fatalf("attrs = %v, want only kept", attrs)
	}
}
