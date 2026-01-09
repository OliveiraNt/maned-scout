// Package utils provides common utilities for the Maned Scout application.
// It includes logging functionality with structured logging support via charmbracelet/log,
// configurable log levels through environment variables, and runtime log level adjustment.
package utils

import (
	"os"
	"strings"
	"sync"

	chlog "github.com/charmbracelet/log"
)

// Logger is the application-wide structured logger.
var (
	Logger *chlog.Logger
	mu     sync.RWMutex
)

const (
	debugLevel = "debug"
	infoLevel  = "info"
	warnLevel  = "warn"
	errorLevel = "error"
)

// InitLogger initializes the global logger with level from MANED_SCOUT_LOG_LEVEL.
// Valid levels: debug, info, warn, errorLevel.
func InitLogger() {
	mu.Lock()
	defer mu.Unlock()
	initLogger()
}

func initLogger() {
	if Logger != nil {
		return
	}
	l := chlog.New(os.Stdout)
	l.SetTimeFormat("2006-01-02 15:04:05.000")
	l.SetReportTimestamp(true)
	levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("MANED_SCOUT_LOG_LEVEL")))
	switch levelStr {
	case debugLevel:
		l.SetLevel(chlog.DebugLevel)
	case warnLevel:
		l.SetLevel(chlog.WarnLevel)
	case errorLevel:
		l.SetLevel(chlog.ErrorLevel)
	default:
		l.SetLevel(chlog.InfoLevel)
	}
	Logger = l
}

// SetLogLevel allows changing level at runtime.
func SetLogLevel(level string) {
	mu.Lock()
	defer mu.Unlock()

	if Logger == nil {
		initLogger()
	}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case debugLevel:
		Logger.SetLevel(chlog.DebugLevel)
	case infoLevel:
		Logger.SetLevel(chlog.InfoLevel)
	case warnLevel:
		Logger.SetLevel(chlog.WarnLevel)
	case errorLevel:
		Logger.SetLevel(chlog.ErrorLevel)
	}
}
