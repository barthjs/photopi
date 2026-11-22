package app_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/media"
)

// testConfig returns the normalized default configuration with its storage in a temporary directory.
func testConfig(t *testing.T) config.Config {
	t.Helper()

	cfg, err := config.Defaults()
	require.NoError(t, err)
	cfg.Storage.Driver = media.DriverLocal
	cfg.Storage.Path = filepath.Join(t.TempDir(), "data")
	cfg.System = config.SystemConfig{}
	require.NoError(t, config.Validate(&cfg))
	return cfg
}
