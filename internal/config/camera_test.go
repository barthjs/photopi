package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/camera"
)

func TestValidateCamera(t *testing.T) {
	t.Parallel()

	t.Run("accepts the rpicam source", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		rpiCam(&cfg)

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, camera.SourceRPiCam, cfg.Camera.Source)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "unknown source",
				change:  func(c *Config) { c.Camera.Source = "film" },
				wantErr: "camera.source",
			},
			{
				name:    "rpicam without capture size",
				change:  func(c *Config) { rpiCam(c); c.Camera.CaptureWidth = 0 },
				wantErr: "camera.capture_width",
			},
			{
				name:    "rpicam with large capture",
				change:  func(c *Config) { rpiCam(c); c.Camera.CaptureWidth = 20000 },
				wantErr: "at most",
			},
			{
				name:    "rpicam without preview size",
				change:  func(c *Config) { rpiCam(c); c.Camera.PreviewHeight = 0 },
				wantErr: "camera.preview_width",
			},
			{
				name:    "rpicam with large preview",
				change:  func(c *Config) { rpiCam(c); c.Camera.PreviewWidth = 5000 },
				wantErr: "at most",
			},
			{
				name:    "rpicam without framerate",
				change:  func(c *Config) { rpiCam(c); c.Camera.PreviewFPS = 0 },
				wantErr: "camera.preview_fps",
			},
			{
				name:    "rpicam with high framerate",
				change:  func(c *Config) { rpiCam(c); c.Camera.PreviewFPS = 500 },
				wantErr: "camera.preview_fps",
			},
			{
				name:    "rpicam with other aspect ratio",
				change:  func(c *Config) { rpiCam(c); c.Camera.PreviewHeight = 600 },
				wantErr: "aspect ratio",
			},
			{
				name:    "exposure time that is no offered value",
				change:  func(c *Config) { rpiCam(c); c.Camera.ExposureTime = "1/7" },
				wantErr: "exposure_time",
			},
			{
				name:    "rpicam with gain below 1",
				change:  func(c *Config) { rpiCam(c); c.Camera.AnalogueGain = 0.5 },
				wantErr: "camera.analogue_gain",
			},
			{
				name:    "rpicam with high gain",
				change:  func(c *Config) { rpiCam(c); c.Camera.AnalogueGain = 64 },
				wantErr: "camera.analogue_gain",
			},
			{
				name:    "rpicam with large exposure compensation",
				change:  func(c *Config) { rpiCam(c); c.Camera.ExposureCompensation = -11 },
				wantErr: "camera.exposure_compensation",
			},
			{
				name:    "rpicam without jpeg quality",
				change:  func(c *Config) { rpiCam(c); c.Camera.JPEGQuality = 0 },
				wantErr: "camera.jpeg_quality",
			},
			{
				name:    "rpicam with high jpeg quality",
				change:  func(c *Config) { rpiCam(c); c.Camera.JPEGQuality = 101 },
				wantErr: "camera.jpeg_quality",
			},
			{
				name:    "unknown white balance",
				change:  func(c *Config) { c.Camera.WhiteBalance = "neon" },
				wantErr: "camera.white_balance",
			},
			{
				name:    "unsupported rotation",
				change:  func(c *Config) { c.Camera.Rotation = 90 },
				wantErr: "camera.rotation",
			},
			{
				name:    "unknown autofocus mode",
				change:  func(c *Config) { c.Camera.AutofocusMode = "macro" },
				wantErr: "camera.autofocus_mode",
			},
			{
				name:    "unknown denoise",
				change:  func(c *Config) { c.Camera.Denoise = "strong" },
				wantErr: "camera.denoise",
			},
			{
				name:    "overlay without png",
				change:  func(c *Config) { c.Camera.OverlayPath = "overlay.jpg" },
				wantErr: "camera.overlay_path",
			},
		})
	})

	t.Run("ignores the image settings of other sources", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Camera.Source = camera.SourceGphoto
		cfg.Camera.JPEGQuality = 0
		cfg.Camera.AnalogueGain = 0.5

		err := Validate(&cfg)

		assert.NoError(t, err)
	})
}

// rpiCam selects the rpicam source with a capture and preview size of the same aspect ratio.
func rpiCam(c *Config) {
	c.Camera.Source = camera.SourceRPiCam
	c.Camera.CaptureWidth, c.Camera.CaptureHeight = 4000, 2400
	c.Camera.PreviewWidth, c.Camera.PreviewHeight = 800, 480
	c.Camera.PreviewFPS = 15
}
