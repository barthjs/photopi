package camera_test

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/camera"
)

// TestWebcamCamera_StreamPreview checks that the backend offers no preview for a webcam,
// since the browser shows the webcam itself.
func TestWebcamCamera_StreamPreview(t *testing.T) {
	t.Parallel()

	t.Run("rejects the preview", func(t *testing.T) {
		t.Parallel()

		cam, err := camera.New(zerolog.Nop(), camera.Options{
			Source: camera.SourceWebcam,
		})
		require.NoError(t, err)
		var out bytes.Buffer

		err = cam.StreamPreview(t.Context(), &out, "frame")

		assert.ErrorIs(t, err, camera.ErrUnsupported)
		assert.Empty(t, out.String())
	})
}

// TestWebcamCamera_Capture checks that the backend takes no pictures for a webcam,
// since the browser takes webcam pictures itself.
func TestWebcamCamera_Capture(t *testing.T) {
	t.Parallel()

	t.Run("rejects the capture", func(t *testing.T) {
		t.Parallel()

		cam, err := camera.New(zerolog.Nop(), camera.Options{
			Source: camera.SourceWebcam,
		})
		require.NoError(t, err)

		still, err := cam.Capture(t.Context())

		assert.ErrorIs(t, err, camera.ErrUnsupported)
		assert.Nil(t, still)
	})
}
