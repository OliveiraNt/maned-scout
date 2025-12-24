package config_test

import (
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/stretchr/testify/require"
)

func TestI18n_Defaults(t *testing.T) {
	t.Parallel()
	// Ensure the types exist and can be referenced; no behavior to test here
	var fc config.FileConfig
	require.NotNil(t, &fc)
}
