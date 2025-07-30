package logger

import (
	"cmp"
	"fmt"
)

type Config struct {
	// ServiceName that will be added to the log (service="service-name") -> default: "unnamed-service"
	ServiceName string `env:"LOGGER_SERVICE_NAME"`
	// Output defines where the log will be written. Options: "stdout", "stderr", "file". It can be multiple values -> default: "stdout"
	Output []Output `env:"LOGGER_OUTPUT"`
	// OutputFilePath defines the path to the log file -> default: "./app.log"
	OutputFilePath string `env:"LOGGER_OUTPUT_FILE_PATH"`
	// OutputFormat defines the format of the log. Options: "logfmt", "logfmt_no_color", "json" -> default: "logfmt_no_color"
	OutputFormat Format `env:"LOGGER_OUTPUT_FORMAT"`
	// StdoutFormat defines the format of the stdout log. If left blank, OutputFormat will be used instead. Options: "logfmt", "logfmt_no_color", "json" -> default: ""
	StdoutFormat Format `env:"LOGGER_STDOUT_FORMAT"`
	// StderrFormat defines the format of the stderr log. If left blank, OutputFormat will be used instead. Options: "logfmt", "logfmt_no_color", "json" -> default: ""
	StderrFormat Format `env:"LOGGER_STDERR_FORMAT"`
	// FileFormat defines the format of the file log. If left blank, OutputFormat will be used instead. Options: "logfmt", "logfmt_no_color", "json" -> default: ""
	FileFormat Format `env:"LOGGER_FILE_FORMAT"`
	// Level defines the level of the log. Options: "disabled", "trace", "debug", "info", "warning", "error", "fatal", "panic" -> default: "debug"
	Level Level `env:"LOGGER_LEVEL"`
	// RotateEnabled defines whether the log rotation is enabled -> default: "true"
	RotateEnabled bool `env:"LOGGER_ROTATE_ENABLED"`
	// RotateMaxSizeMB defines the maximum size of the log file in MB -> default: "10"
	RotateMaxSizeMB int `env:"LOGGER_ROTATE_MAX_SIZE_MB"`
	// RotateMaxBackups defines the maximum number of log files to keep -> default: "3"
	RotateMaxBackups int `env:"LOGGER_ROTATE_MAX_BACKUPS"`
	// RotateMaxAgeDays defines the maximum age of the log files in days -> default: "28"
	RotateMaxAgeDays int `env:"LOGGER_ROTATE_MAX_AGE_DAYS"`
	// RotateCompress defines whether the log files should be compressed -> default: "true"
	RotateCompress bool `env:"LOGGER_ROTATE_COMPRESS"`
	// InjectTraceInfo defines whether the trace and span ID should be injected into the log -> default: "true"
	InjectTraceInfo bool `env:"LOGGER_INJECT_TRACE_INFO"`

	// for internal use
	parseEnv bool
}

func DefaultConfig() *Config {
	return &Config{}
}

func ParseLoggerEnvVar() *Config {
	return &Config{
		parseEnv: true,
	}
}

func (c *Config) setDefaults() {
	c.ServiceName = cmp.Or(c.ServiceName, "unnamed-service")
	if c.Output == nil {
		c.Output = []Output{OutputStdout}
	}
	c.OutputFilePath = cmp.Or(c.OutputFilePath, "./app.log")
	c.OutputFormat = cmp.Or(c.OutputFormat, FormatLogfmtNoColor)
	c.Level = cmp.Or(c.Level, LevelDebug)
	c.RotateEnabled = cmp.Or(c.RotateEnabled, true)
	c.RotateMaxSizeMB = cmp.Or(c.RotateMaxSizeMB, 10)
	c.RotateMaxBackups = cmp.Or(c.RotateMaxBackups, 3)
	c.RotateMaxAgeDays = cmp.Or(c.RotateMaxAgeDays, 28)
	c.RotateCompress = cmp.Or(c.RotateCompress, true)
	c.InjectTraceInfo = cmp.Or(c.InjectTraceInfo, true)
}

func (c *Config) validate() error {
	// Validate Output
	for _, op := range c.Output {
		switch op {
		case OutputStdout, OutputStderr, OutputFile:
		default:
			return fmt.Errorf("invalid output: %s, valid values: stdout, stderr, file", op)
		}
	}

	// Validate OutputFormat
	switch c.OutputFormat {
	case FormatLogfmt, FormatLogfmtNoColor, FormatJSON:
	default:
		return fmt.Errorf("invalid output format: %s, valid values: logfmt, logfmt_no_color, json", c.OutputFormat)
	}

	// Validate StdoutFormat
	switch c.StdoutFormat {
	case FormatLogfmt, FormatLogfmtNoColor, FormatJSON, "":
	default:
		return fmt.Errorf("invalid stdout format: %s, valid values: logfmt, logfmt_no_color, json, empty", c.StdoutFormat)
	}

	// Validate StderrFormat
	switch c.StderrFormat {
	case FormatLogfmt, FormatLogfmtNoColor, FormatJSON, "":
	default:
		return fmt.Errorf("invalid stderr format: %s, valid values: logfmt, logfmt_no_color, json, empty", c.StderrFormat)
	}

	// Validate FileFormat
	switch c.FileFormat {
	case FormatLogfmt, FormatLogfmtNoColor, FormatJSON, "":
	default:
		return fmt.Errorf("invalid file format: %s, valid values: logfmt, logfmt_no_color, json, empty", c.FileFormat)
	}

	// Validate Level
	switch c.Level {
	case LevelDisabled, LevelTrace, LevelDebug, LevelInfo, LevelWarning, LevelError, LevelFatal, LevelPanic:
	default:
		return fmt.Errorf("invalid level: %s, valid values: disabled, trace, debug, info, warning, error, fatal, panic", c.Level)
	}

	// Validate RotateMaxSizeMB
	if c.RotateMaxSizeMB < 1 {
		return fmt.Errorf("invalid rotate max size mb: %d, valid values: >= 1", c.RotateMaxSizeMB)
	}

	// Validate RotateMaxBackups
	if c.RotateMaxBackups < 1 {
		return fmt.Errorf("invalid rotate max backups: %d, valid values: >= 1", c.RotateMaxBackups)
	}

	// Validate RotateMaxAgeDays
	if c.RotateMaxAgeDays < 1 {
		return fmt.Errorf("invalid rotate max age days: %d, valid values: >= 1", c.RotateMaxAgeDays)
	}

	return nil
}
