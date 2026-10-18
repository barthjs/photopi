package share_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/share"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		driver  share.Driver
		wantNil bool
		wantErr string
	}{
		{
			name:   "returns a provider that disables sharing",
			driver: share.DriverNone,
		},
		{
			name:   "returns the nextcloud provider",
			driver: share.DriverNextcloud,
		},
		{
			name:   "returns the self provider",
			driver: share.DriverSelf,
		},
		{
			name:    "rejects an unknown driver",
			driver:  "carrier-pigeon",
			wantNil: true,
			wantErr: `unknown driver "carrier-pigeon"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, err := share.New(share.Options{
				Driver: tt.driver,
			})

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantNil, provider == nil)
		})
	}
}

func TestEnabled(t *testing.T) {
	t.Parallel()

	t.Run("reports a provider as enabled", func(t *testing.T) {
		t.Parallel()

		nextcloud, err := share.New(share.Options{
			Driver: share.DriverNextcloud,
		})
		require.NoError(t, err)

		enabled := share.Enabled(nextcloud)

		assert.True(t, enabled)
	})

	t.Run("reports the provider without sharing as disabled", func(t *testing.T) {
		t.Parallel()

		provider, err := share.New(share.Options{
			Driver: share.DriverNone,
		})
		require.NoError(t, err)

		enabled := share.Enabled(provider)

		assert.False(t, enabled)
	})

	t.Run("reports a missing provider as disabled", func(t *testing.T) {
		t.Parallel()

		enabled := share.Enabled(nil)

		assert.False(t, enabled)
	})
}

func TestNoneProvider_TestConnection(t *testing.T) {
	t.Parallel()

	t.Run("succeeds without anything to test", func(t *testing.T) {
		t.Parallel()

		provider, err := share.New(share.Options{
			Driver: share.DriverNone,
		})
		require.NoError(t, err)

		err = provider.TestConnection(t.Context())

		assert.NoError(t, err, "there is nothing to test")
	})
}

func TestNoneProvider_Upload(t *testing.T) {
	t.Parallel()

	t.Run("rejects the upload", func(t *testing.T) {
		t.Parallel()

		provider, err := share.New(share.Options{
			Driver: share.DriverNone,
		})
		require.NoError(t, err)

		shareURL, err := provider.Upload(t.Context(), "run", []share.File{
			{
				Name: "a.jpg",
			},
		})

		require.ErrorIs(t, err, share.ErrUnsupported)
		assert.Empty(t, shareURL)
	})
}
