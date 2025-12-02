package registry

import (
	"os"
	"strings"

	chlog "github.com/charmbracelet/log"
)

// Logger is the application-wide structured logger.
var Logger *chlog.Logger

// InitLogger initializes the global logger with level from KDASH_LOG_LEVEL.
// Valid levels: debug, info, warn, error.
func InitLogger() {
	if Logger != nil {
		return
	}
	l := chlog.New(os.Stdout)
	l.SetTimeFormat(chlog.DefaultTimeFormat)
	l.SetReportTimestamp(true)
	// derive level from env
	levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("KDASH_LOG_LEVEL")))
	switch levelStr {
	case "debug":
		l.SetLevel(chlog.DebugLevel)
	case "warn":
		l.SetLevel(chlog.WarnLevel)
	case "error":
		l.SetLevel(chlog.ErrorLevel)
	default:
		// info is default
		l.SetLevel(chlog.InfoLevel)
	}
	Logger = l
}

// SetLogLevel allows changing level at runtime.
func SetLogLevel(level string) {
	if Logger == nil {
		InitLogger()
	}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		Logger.SetLevel(chlog.DebugLevel)
	case "info":
		Logger.SetLevel(chlog.InfoLevel)
	case "warn":
		Logger.SetLevel(chlog.WarnLevel)
	case "error":
		Logger.SetLevel(chlog.ErrorLevel)
	}
}
