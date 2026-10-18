package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestValidateSystem(t *testing.T) {
	t.Parallel()

	t.Run("trims the commands", func(t *testing.T) {
		t.Parallel()

		command := apitest.Faker(t).Word()
		cfg := testConfig(t)
		cfg.System.KioskStopCommand = "  " + command + "  "

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, command, cfg.System.KioskStopCommand)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "long command",
				change:  func(c *Config) { c.System.AllStopCommand = strings.Repeat("a", 1025) },
				wantErr: "system.all_stop_command",
			},
		})
	})
}
