package utils

import (
	"os"
	"strings"

	chlog "github.com/charmbracelet/log"
)

// Logger is the application-wide structured logger.
var Logger *chlog.Logger

// InitLogger initializes the global logger with level from MANED_SCOUT_LOG_LEVEL.
// Valid levels: debug, info, warn, error.
func InitLogger() {
	if Logger != nil {
		return
	}
	l := chlog.New(os.Stdout)
	l.SetTimeFormat("2006-01-02 15:04:05.000")
	l.SetReportTimestamp(true)
	levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("MANED_SCOUT_LOG_LEVEL")))
	switch levelStr {
	case "debug":
		l.SetLevel(chlog.DebugLevel)
	case "warn":
		l.SetLevel(chlog.WarnLevel)
	case "error":
		l.SetLevel(chlog.ErrorLevel)
	default:
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
