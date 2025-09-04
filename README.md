# ksckaan1/logger

[![tag](https://img.shields.io/github/release/ksckaan1/logger.svg)](https://github.com/ksckaan1/logger/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-%23007d9c)
[![Go report](https://goreportcard.com/badge/github.com/ksckaan1/logger)](https://goreportcard.com/report/github.com/ksckaan1/logger)
[![Contributors](https://img.shields.io/github/contributors/ksckaan1/logger)](https://github.com/ksckaan1/logger/graphs/contributors)
[![LICENSE](https://img.shields.io/badge/LICENCE-MIT-orange?style=flat)](./LICENSE)

A simple, fast, and highly configurable Go logging package. This package leverages zerolog for high-performance logging, lumberjack for log file rotation, and can be integrated with OpenTelemetry for distributed tracing.

## Features
- **Flexible Output Destinations:** Write logs to `stdout`, `stderr`, and/or `files`, individually or simultaneously.
- **Log Rotation:** When writing to files, automatically handles log file rotation, compression, and retention using `lumberjack`.
- **Customizable Output Formats:** Each output destination can have its own format: `logfmt`, `logfmt without color`, or `json`.
- **Nested Loggers:** Create hierarchical logger structures using the Sub method for more organized and contextual logging.
- **OpenTelemetry Integration:** Easily embed Trace and Span IDs into your logs for better observability with OpenTelemetry.
- **Configurable Log Levels:** Define the logging level (e.g., `info`, `debug`, `error`) for your logger.

## Installation

To add the package to your project, use the following command:

```sh
go get github.com/ksckaan1/logger
```

## Usage

Import the package:

```go
import "github.com/ksckaan1/logger"
```

### Create a logger

- With default config:

  ```go
  lg, err := logger.New(logger.DefaultConfig())
  if err != nil {
    panic(err)
  }
  defer lg.Close()
  ```

- Using environment variables:

  ```go
  lg, err := logger.New(logger.ParseLoggerEnvVar())
  if err != nil {
    panic(err)
  }
  defer lg.Close()
  ```

- Or with custom config:

  ```go
  lg, err := logger.New(&logger.Config{
    ServiceName: "my-service",
    // ...
  })
  if err != nil {
    panic(err)
  }
  defer lg.Close()
  ```

### Print Logs

```go
ctx := context.Background()

lg.Trace(ctx, "this is a trace message")
lg.Debug(ctx, "this is a debug message")
lg.Info(ctx, "this is an info message")
lg.Warn(ctx, "this is a warning message")
lg.Error(ctx, "this is an error message")
lg.Fatal(ctx, "this is a fatal message")
lg.Panic(ctx, "this is a panic message")
```

**Example output:**

```logfmt
2025-07-31T02:23:17+03:00 TRC logger_test.go:37 > this is a trace message service=my-service
2025-07-31T02:23:17+03:00 DBG logger_test.go:38 > this is a debug message service=my-service
2025-07-31T02:23:17+03:00 INF logger_test.go:39 > this is an info message service=my-service
2025-07-31T02:23:17+03:00 WRN logger_test.go:40 > this is a warning message service=my-service
2025-07-31T02:23:17+03:00 ERR logger_test.go:41 > this is an error message service=my-service
2025-07-31T02:23:17+03:00 FAT logger_test.go:42 > this is a fatal message service=my-service
2025-07-31T02:23:17+03:00 PAN logger_test.go:43 > this is a panic message service=my-service
```

### Adding Extra Fields

```go
ctx := context.Background()

lg.Info(ctx, "this is an info message",
	"key1", "value 1",
	"key2", 42,
	"key3", true,
)
```

**Example output:**

```logfmt
2025-07-31T02:23:17+03:00 INF logger_test.go:39 > this is an info message key1="value 1" key2=42 key3=true service=my-service
```

### Create Sub Logger

```go
lg, err := logger.New(&logger.Config{
  ServiceName: "my-service",
  // ...
})
if err != nil {
  panic(err)
}

ctx := context.Background()

sublg := lg.Sub("sub-service")
sublg.Info(ctx, "this is an info message")
```

**Example output:**

```logfmt
2025-07-31T02:23:17+03:00 INF logger_test.go:39 > this is an info message service=my-service/sub-service
```

### Configuration And Environment Variables

| Config           | Description                                   | Options                                                                  | Default                    | Environment variable         |
| ---------------- | --------------------------------------------- | ------------------------------------------------------------------------ | -------------------------- | ---------------------------- |
| ServiceName      | Service name that will be added to the log    | can be any string                                                        | `unnamed-service`          | `LOGGER_SERVICE_NAME`        |
| Output           | Output defines where the log will be written. | `stdout`, `stderr`, `file`. It can be multiple values separated by comma | `stdout`                   | `LOGGER_OUTPUT`              |
| OutputFilePath   | Output file path                              | string                                                                   | `./app.log`                | `LOGGER_OUTPUT_FILE_PATH`    |
| OutputFormat     | Output format.                                | `logfmt`, `logfmt_no_color`, `json`                                      | `logfmt_no_color`          | `LOGGER_OUTPUT_FORMAT`       |
| StdoutFormat     | Stdout format.                                | `logfmt`, `logfmt_no_color`, `json`                                      | inherits from OutputFormat | `LOGGER_STDOUT_FORMAT`       |
| StderrFormat     | Stderr format.                                | `logfmt`, `logfmt_no_color`, `json`                                      | inherits from OutputFormat | `LOGGER_STDERR_FORMAT`       |
| FileFormat       | File format.                                  | `logfmt`, `logfmt_no_color`, `json`                                      | inherits from OutputFormat | `LOGGER_FILE_FORMAT`         |
| Level            | Log level.                                    | `LOGGER_LEVEL`                                                           |
| RotateEnabled    | Log rotation enabled                          | `true`, `false`                                                          | `true`                     | `LOGGER_ROTATE_ENABLED`      |
| RotateMaxSizeMB  | Log rotation max size in MB                   | > 0                                                                      | `10`                       | `LOGGER_ROTATE_MAX_SIZE_MB`  |
| RotateMaxBackups | Log rotation max backups                      | > 0                                                                      | `3`                        | `LOGGER_ROTATE_MAX_BACKUPS`  |
| RotateMaxAgeDays | Log rotation max age in days                  | > 0                                                                      | `28`                       | `LOGGER_ROTATE_MAX_AGE_DAYS` |
| RotateCompress   | Log rotation compress                         | `true`, `false`                                                          | `true`                     | `LOGGER_ROTATE_COMPRESS`     |
| InjectTraceInfo  | Inject trace info                             | `true`, `false`                                                          | `true`                     | `LOGGER_INJECT_TRACE_INFO`   |
