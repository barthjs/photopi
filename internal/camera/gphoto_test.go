//go:build unix

package camera_test

import (
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/camera"
)

// TestGphotoCam_StreamPreview is not parallel, since the fake tools replace PATH.
func TestGphotoCam_StreamPreview(t *testing.T) {
	t.Run("streams the frames of the live view", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.gphoto(t, 3)
		cam, _ := newGphotoCam(t)
		out := &syncBuffer{}

		err := cam.StreamPreview(t.Context(), out, "live")

		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 3, strings.Count(out.String(), "--live\r\n"))
		assert.Equal(t, "--capture-movie --stdout", tools.args(t, "gphoto2"))
	})

	t.Run("logs why the live view ended", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.gphoto(t, 1)
		tools.flag(t, "gphoto2.fail")
		cam, logs := newGphotoCam(t)

		err := cam.StreamPreview(t.Context(), io.Discard, "live")

		require.ErrorIs(t, err, io.EOF)
		assert.Contains(t, logs.String(), "gphoto2 live view exited")
		assert.Contains(t, logs.String(), "device busy")
	})
}

// TestGphotoCam_Capture is not parallel, since the fake tools replace PATH.
func TestGphotoCam_Capture(t *testing.T) {
	t.Run("returns the still", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.gphoto(t, 1)
		cam, logs := newGphotoCam(t)

		data, err := cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Equal(t, still, string(*data))
		assert.Equal(t, "--capture-image-and-download --stdout", tools.args(t, "gphoto2"))
		assert.Empty(t, tools.env(t, "gphoto2"), "only the rpicam tools get libcamera settings")
		assert.Contains(t, logs.String(), `"still_ms":`)
		assert.NotContains(t, logs.String(), `"level":"warn"`)
	})

	t.Run("logs warnings", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.gphoto(t, 1)
		tools.flag(t, "gphoto2.warn")
		cam, logs := newGphotoCam(t)

		_, err := cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Contains(t, logs.String(), "gphoto2 capture reported warnings")
	})

	t.Run("stops the running preview", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.gphoto(t, 0)
		cam, _ := newGphotoCam(t)
		out := &syncBuffer{}
		streamErr := make(chan error, 1)
		go func() {
			streamErr <- cam.StreamPreview(t.Context(), out, "live")
		}()
		require.Eventually(t, func() bool { return strings.Contains(out.String(), "--live") }, 5*time.Second, 10*time.Millisecond)

		data, err := cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Equal(t, still, string(*data))
		select {
		case err := <-streamErr:
			assert.ErrorIs(t, err, camera.ErrPreviewStopped)
		case <-time.After(5 * time.Second):
			t.Fatal("the live view kept running after the capture")
		}
	})

	t.Run("returns the error of a failing capture", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.gphoto(t, 1)
		tools.flag(t, "gphoto2.fail")
		cam, _ := newGphotoCam(t)

		data, err := cam.Capture(t.Context())

		require.Error(t, err)
		assert.ErrorContains(t, err, "gphoto2 capture failed")
		assert.ErrorContains(t, err, "device busy")
		assert.Nil(t, data)
	})
}

// gphoto writes a fake gphoto2 that prints frames for the live view and a still for a capture.
// gphoto2 serves both with one program, so the fake decides by its arguments.
func (f *fakeTools) gphoto(t *testing.T, frames int) {
	t.Helper()

	loop := "while true; do printf '" + frame + "'; " + f.sleep + " 0.02; done"
	if frames > 0 {
		loop = "i=0; while [ $i -lt " + strconv.Itoa(frames) + " ]; do printf '" + frame + "'; i=$((i+1)); done"
	}
	f.script(t, "gphoto2", `case "$1" in
--capture-movie) trap 'exit 0' TERM; `+loop+`;;
--capture-image-and-download) printf '\377\330still\377\331';;
*) exit 2;;
esac
`)
}

// newGphotoCam builds a gphoto2 camera that logs to the returned buffer.
func newGphotoCam(t *testing.T) (camera.Camera, *syncBuffer) {
	t.Helper()

	logger, logs := newLogger()
	cam, err := camera.New(logger, camera.Options{
		Source: camera.SourceGphoto,
	})
	require.NoError(t, err)
	return cam, logs
}
