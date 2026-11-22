package app_test

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/app"
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
