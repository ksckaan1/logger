package logger

import (
	"github.com/rs/zerolog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

// otelHook bridges zerolog events to the global OpenTelemetry logger provider.
// Replaces go.opentelemetry.io/contrib/bridges/otelzerolog, which was removed
// upstream and never migrated to the otel/log v1 API.
type otelHook struct {
	logger log.Logger
}

func newOtelHook(name string) zerolog.Hook {
	return otelHook{logger: global.GetLoggerProvider().Logger(name)}
}

func (h otelHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	var r log.Record
	r.SetSeverity(otelSeverity(level))
	r.SetSeverityText(level.String())
	r.SetBody(attribute.StringValue(msg))
	h.logger.Emit(e.GetCtx(), r)
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
