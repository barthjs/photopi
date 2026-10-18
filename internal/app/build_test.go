package app_test

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/app"
	"github.com/barthjs/photopi/internal/update"
)

func TestNewStores(t *testing.T) {
	t.Parallel()

	t.Run("builds the stores on the storage", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		storage, err := app.NewStorage(zerolog.Nop(), cfg)
		require.NoError(t, err)

		stores := app.NewStores(storage)

		require.NotNil(t, stores.Capture)
		require.NotNil(t, stores.Collage)
		require.NotNil(t, stores.Overlay)
		require.NotNil(t, stores.Run)
		require.NotNil(t, stores.Layout)
		layouts, err := stores.Layout.List(t.Context())
		require.NoError(t, err)
		assert.Empty(t, layouts, "a fresh storage holds no layouts")
	})
}

func TestNewStorage(t *testing.T) {
	t.Parallel()

	t.Run("uses the configured path", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)

		storage, err := app.NewStorage(zerolog.Nop(), cfg)

		require.NoError(t, err)
		require.NoError(t, storage.TestConnection(t.Context()))
		assert.DirExists(t, cfg.Storage.Path)
	})
}

func TestNewUpdater(t *testing.T) {
	t.Parallel()

	t.Run("rejects installs of a development build", func(t *testing.T) {
		t.Parallel()

		updater := app.NewUpdater("dev")

		assert.ErrorIs(t, updater.Supported(), update.ErrDevelopmentBuild)
	})
}
