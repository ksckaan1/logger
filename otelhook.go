package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
)

// otelEmitter sends log records to the global OpenTelemetry logger provider.
//
// A zerolog.Hook cannot forward the log fields: a hook only gets the message,
// the level and the *zerolog.Event, and zerolog has no API to read back the
// fields already appended to an event (rs/zerolog#493). The fields are
// therefore forwarded from Logger.log, where they are still available as
// key/value pairs.
type otelEmitter struct {
	logger log.Logger
}

func newOtelEmitter(name string) *otelEmitter {
	// otel.Logger is updated in place once a provider is registered, so this
	// works even when the logger is created before the provider.
	return &otelEmitter{logger: otel.Logger(name)}
}

func (h *otelEmitter) emit(ctx context.Context, level zerolog.Level, message string, attrs []attribute.KeyValue) {
	var r log.Record
	r.SetSeverity(otelSeverity(level))
	r.SetSeverityText(level.String())
	r.SetBody(attribute.StringValue(message))
	r.AddAttributes(attrs...)
	h.logger.Emit(ctx, r)
}

func otelSeverity(level zerolog.Level) log.Severity {
	switch level {
	case zerolog.DebugLevel:
		return log.SeverityDebug
	case zerolog.InfoLevel:
		return log.SeverityInfo
	case zerolog.WarnLevel:
		return log.SeverityWarn
	case zerolog.ErrorLevel:
		return log.SeverityError
	case zerolog.PanicLevel:
		return log.SeverityFatal1
	case zerolog.FatalLevel:
		return log.SeverityFatal2
	default:
		return log.SeverityUndefined
	}
}

// otelAttrs mirrors zerolog's own field handling: alternating key/value pairs,
// where a non-string key or a trailing unpaired value is dropped.
func otelAttrs(fields []any) []attribute.KeyValue {
	if n := len(fields); n&0x1 == 1 {
		fields = fields[:n-1]
	}

	attrs := make([]attribute.KeyValue, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		if kv, ok := otelAttr(key, fields[i+1]); ok {
			attrs = append(attrs, kv)
		}
	}
	return attrs
}

func otelAttr(key string, v any) (attribute.KeyValue, bool) {
	switch val := v.(type) {
	case nil:
		return attribute.KeyValue{}, false
	case string:
		return attribute.String(key, val), true
	case []byte:
		return attribute.String(key, string(val)), true
	case bool:
		return attribute.Bool(key, val), true
	case int:
		return attribute.Int64(key, int64(val)), true
	case int8:
		return attribute.Int64(key, int64(val)), true
	case int16:
		return attribute.Int64(key, int64(val)), true
	case int32:
		return attribute.Int64(key, int64(val)), true
	case int64:
		return attribute.Int64(key, val), true
	case uint:
		return attribute.Int64(key, int64(val)), true
	case uint8:
		return attribute.Int64(key, int64(val)), true
	case uint16:
		return attribute.Int64(key, int64(val)), true
	case uint32:
		return attribute.Int64(key, int64(val)), true
	case uint64:
		return attribute.Int64(key, int64(val)), true
	case float32:
		return attribute.Float64(key, float64(val)), true
	case float64:
		return attribute.Float64(key, val), true
	case time.Time:
		return attribute.String(key, val.Format(time.RFC3339Nano)), true
	case time.Duration:
		return attribute.String(key, val.String()), true
	case error:
		return attribute.String(key, val.Error()), true
	case []string:
		return attribute.StringSlice(key, val), true
	case []int:
		return attribute.Int64Slice(key, toInt64s(val)), true
	case []int64:
		return attribute.Int64Slice(key, val), true
	case []float64:
		return attribute.Float64Slice(key, val), true
	case []bool:
		return attribute.BoolSlice(key, val), true
	case fmt.Stringer:
		return attribute.String(key, val.String()), true
	default:
		// shortcut: nested objects and other types become a formatted string
		// instead of a structured attribute; add a case above when needed.
		return attribute.String(key, fmt.Sprintf("%+v", val)), true
	}
}

func toInt64s(v []int) []int64 {
	out := make([]int64, len(v))
	for i, n := range v {
		out[i] = int64(n)
	}
	return out
}
