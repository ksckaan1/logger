# ksckaan1/logger

A simple, fast, and highly configurable Go logging package. This package leverages zerolog for high-performance logging, lumberjack for log file rotation, and can be integrated with OpenTelemetry for distributed tracing.

## Features
- **Flexible Output Destinations:** Write logs to `stdout`, `stderr`, and/or `files`, individually or simultaneously.
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
  ```

- Using environment variables:

  ```go
  lg, err := logger.New(logger.ParseLoggerEnvVar())
  if err != nil {
    panic(err)
  }
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








