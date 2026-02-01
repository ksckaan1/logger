package logger

var Default *Logger = nil

func init() {
	Default, _ = New(DefaultConfig())
}

func SetDefault(logger *Logger) {
	Default = logger
}
