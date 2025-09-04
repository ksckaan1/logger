package logger_test

import (
	"context"
	"os"

	"testing"

	"github.com/ksckaan1/logger"
)

func TestLogger_WithEnv(t *testing.T) {
	os.Setenv("LOGGER_OUTPUT", "stdout,file")
	os.Setenv("LOGGER_OUTPUT_FORMAT", "logfmt")
	os.Setenv("LOGGER_STDOUT_FORMAT", "logfmt_no_color")
	os.Setenv("LOGGER_STDERR_FORMAT", "logfmt")
	os.Setenv("LOGGER_FILE_FORMAT", "json")
	os.Setenv("LOGGER_LEVEL", "debug")
	os.Setenv("LOGGER_SERVICE_NAME", "test")
	os.Setenv("LOGGER_OUTPUT_FILE_PATH", "./app.log")

	ctx := context.Background()
	lg, err := logger.New(logger.ParseLoggerEnvVar())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()
	lg.Info(ctx, "test")
	lg.Debug(ctx, "test", "key", "value")
}

func TestLogger_WithDefaultConfig(t *testing.T) {
	ctx := context.Background()
	lg, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()
	lg.Info(ctx, "test")
	lg.Debug(ctx, "test", "key", "value")
}
