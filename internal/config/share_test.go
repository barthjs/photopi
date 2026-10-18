package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/share"
)

func TestValidateShare(t *testing.T) {
	t.Parallel()

	t.Run("trims the base url", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Share.Cloud.Driver = share.DriverSelf
		cfg.Share.Cloud.Self.BaseURL = "https://example.com/"

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, "https://example.com", cfg.Share.Cloud.Self.BaseURL)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "long subject",
				change:  func(c *Config) { c.Share.Email.Subject = strings.Repeat("a", 256) },
				wantErr: "share.email.subject",
			},
			{
				name:    "unknown cloud driver",
				change:  func(c *Config) { c.Share.Cloud.Driver = "ftp" },
				wantErr: "share.cloud.driver",
			},
			{
				name:    "nextcloud without url",
				change:  func(c *Config) { c.Share.Cloud.Driver = share.DriverNextcloud },
				wantErr: "SHARE_CLOUD_NEXTCLOUD_URL",
			},
			{
				name:    "self without base url",
				change:  func(c *Config) { c.Share.Cloud.Driver = share.DriverSelf },
				wantErr: "SHARE_CLOUD_SELF_BASE_URL",
			},
		})
	})
}
