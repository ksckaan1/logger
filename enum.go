package logger

type Output = string

const (
	OutputStdout Output = "stdout"
	OutputStderr Output = "stderr"
	OutputFile   Output = "file"
)

type Format = string

const (
	FormatLogfmt        Format = "logfmt"
	FormatLogfmtNoColor Format = "logfmt_no_color"
	FormatJSON          Format = "json"
)

type Level = string

const (
	LevelDisabled Level = "disabled"
	LevelTrace    Level = "trace"
	LevelDebug    Level = "debug"
	LevelInfo     Level = "info"
	LevelWarning  Level = "warning"
	LevelError    Level = "error"
	LevelFatal    Level = "fatal"
	LevelPanic    Level = "panic"
)
