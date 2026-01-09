package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitLogger_IdempotentAndDefaultLevel(t *testing.T) {
	_ = os.Unsetenv("MANED_SCOUT_LOG_LEVEL")
	Logger = nil
	InitLogger()
	first := Logger
	require.NotNil(t, first)

	// second call should not recreate
	InitLogger()
	require.Equal(t, first, Logger)
}

func TestSetLogLevel_NoPanics(t *testing.T) {
	Logger = nil
	InitLogger()

	require.NotPanics(t, func() { SetLogLevel("debug") })
	require.NotPanics(t, func() { SetLogLevel("info") })
	require.NotPanics(t, func() { SetLogLevel("warn") })
	require.NotPanics(t, func() { SetLogLevel("errorLevel") })
}
