package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestValidateCapture(t *testing.T) {
	t.Parallel()

	t.Run("trims the event name", func(t *testing.T) {
		t.Parallel()

		eventName := apitest.Faker(t).Word()
		cfg := testConfig(t)
		cfg.Capture.EventName = " " + eventName + " "

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, eventName, cfg.Capture.EventName)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "no images per run",
				change:  func(c *Config) { c.Capture.ImagesPerRun = 0 },
				wantErr: "capture.images_per_run",
			},
			{
				name:    "too many images per run",
				change:  func(c *Config) { c.Capture.ImagesPerRun = 21 },
				wantErr: "capture.images_per_run",
			},
			{
				name:    "empty file prefix",
				change:  func(c *Config) { c.Capture.FilePrefix = " " },
				wantErr: "capture.file_prefix",
			},
			{
				name:    "file prefix with separator",
				change:  func(c *Config) { c.Capture.FilePrefix = "a/b" },
				wantErr: "capture.file_prefix",
			},
			{
				name:    "empty event name",
				change:  func(c *Config) { c.Capture.EventName = "" },
				wantErr: "capture.event_name",
			},
			{
				name:    "event name with separator",
				change:  func(c *Config) { c.Capture.EventName = `a\b` },
				wantErr: "capture.event_name",
			},
			{
				name:    "long welcome message",
				change:  func(c *Config) { c.Capture.WelcomeMessage = strings.Repeat("a", 71) },
				wantErr: "capture.welcome_message",
			},
		})
	})
}
