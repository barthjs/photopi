//go:build unix

package camera_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/camera"
)

// rpiOptions are the camera options of the Raspberry Pi camera tests.
var rpiOptions = camera.Options{
	Source:        camera.SourceRPiCam,
	CaptureWidth:  4056,
	CaptureHeight: 3040,
	PreviewWidth:  1280,
	PreviewHeight: 720,
	PreviewFPS:    25,
}

// TestNewRPiCam is not parallel, since the fake tools replace PATH.
func TestNewRPiCam(t *testing.T) {
	t.Run("rejects a missing still tool", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)

		_, err := camera.New(zerolog.Nop(), rpiOptions)

		require.Error(t, err)
		assert.ErrorContains(t, err, `required tool "rpicam-still" not found`)
	})
}

// TestRPiCam_StreamPreview is not parallel, since the fake tools replace PATH.
func TestRPiCam_StreamPreview(t *testing.T) {
	t.Run("streams the frames of the tool", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 2)
		tools.still(t, "rpicam-still")
		cam, _ := newRPiCam(t)
		out := &syncBuffer{}

		err := cam.StreamPreview(t.Context(), out, "frame")

		require.ErrorIs(t, err, io.EOF, "the stream ends when the tool stops producing frames")
		assert.Equal(t, 2, strings.Count(out.String(), "--frame\r\nContent-Type: image/jpeg"))
		assert.Equal(t, "--nopreview --codec mjpeg --width 1280 --height 720 --framerate 25 --timeout 0 --flush -o -", tools.args(t, "rpicam-vid"))
		assert.Equal(t, "*:WARN", tools.env(t, "rpicam-vid"))
	})

	t.Run("stops with the client", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 0)
		tools.still(t, "rpicam-still")
		cam, _ := newRPiCam(t)
		ctx, cancel := context.WithCancel(t.Context())
		out := &syncBuffer{}
		streamErr := make(chan error, 1)
		go func() {
			streamErr <- cam.StreamPreview(ctx, out, "frame")
		}()
		require.Eventually(t, func() bool { return strings.Contains(out.String(), "--frame") }, 5*time.Second, 10*time.Millisecond)

		cancel()

		select {
		case err := <-streamErr:
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(5 * time.Second):
			t.Fatal("the preview kept running after the client left")
		}
	})

	t.Run("stops when the client stops reading", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 0)
		tools.still(t, "rpicam-still")
		cam, logs := newRPiCam(t)
		started := time.Now()

		err := cam.StreamPreview(t.Context(), failingClient{}, "frame")

		require.Error(t, err)
		assert.Less(t, time.Since(started), 2*time.Second, "the tool exits on SIGTERM before it has to be killed")
		assert.NotContains(t, logs.String(), `"level":"warn"`, "a failing client is no camera problem")
		// The device is free for the next capture
		_, err = cam.Capture(t.Context())
		require.NoError(t, err)
	})

	t.Run("logs why a failing tool ended", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.flag(t, "rpicam-vid.fail")
		tools.still(t, "rpicam-still")
		cam, logs := newRPiCam(t)

		err := cam.StreamPreview(t.Context(), io.Discard, "frame")

		require.ErrorIs(t, err, io.EOF)
		assert.Contains(t, logs.String(), "rpicam-vid exited", "the reason the preview ended is logged")
		assert.Contains(t, logs.String(), "device busy")
		assert.Contains(t, logs.String(), `"level":"warn"`)
	})
}

// TestRPiCam_Capture is not parallel, since the fake tools replace PATH.
func TestRPiCam_Capture(t *testing.T) {
	t.Run("returns the still", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.still(t, "rpicam-still")
		cam, logs := newRPiCam(t)

		data, err := cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Equal(t, still, string(*data))
		assert.Equal(t, "--nopreview --immediate --encoding jpg --thumb none --width 4056 --height 3040 -o -", tools.args(t, "rpicam-still"))
		assert.Equal(t, "*:WARN", tools.env(t, "rpicam-still"), "libcamera only logs warnings")
		assert.Contains(t, logs.String(), `"level":"info"`)
		assert.Contains(t, logs.String(), `"preview_stop_ms":`, "the capture logs how long stopping the preview took")
		assert.Contains(t, logs.String(), `"still_ms":`, "the capture logs how long the still took")
		assert.NotContains(t, logs.String(), `"level":"warn"`)
	})

	t.Run("passes the image settings to the still and the preview", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.still(t, "rpicam-still")
		options := rpiOptions
		options.ExposureTime = 8 * time.Millisecond
		options.AnalogueGain = 2.5
		options.ExposureCompensation = -0.5
		options.WhiteBalance = camera.WhiteBalanceDaylight
		options.HorizontalFlip = true
		options.VerticalFlip = true
		options.Rotation = 180
		options.JPEGQuality = 90
		options.AutofocusMode = camera.AutofocusModeContinuous
		options.Denoise = camera.DenoiseCDNFast
		cam, err := camera.New(zerolog.Nop(), options)
		require.NoError(t, err)

		_, err = cam.Capture(t.Context())
		require.NoError(t, err)
		require.ErrorIs(t, cam.StreamPreview(t.Context(), io.Discard, "frame"), io.EOF)

		image := "--shutter 8000 --gain 2.5 --ev -0.5 --awb daylight --hflip --vflip --rotation 180 --denoise cdn_fast --autofocus-mode continuous"
		assert.Equal(
			t,
			"--nopreview --immediate --encoding jpg --thumb none --width 4056 --height 3040 --quality 90 "+image+" -o -",
			tools.args(t, "rpicam-still"),
		)
		assert.Equal(
			t,
			"--nopreview --codec mjpeg --width 1280 --height 720 --framerate 25 --timeout 0 --flush "+image+" -o -",
			tools.args(t, "rpicam-vid"),
			"the preview looks like the still",
		)
	})

	t.Run("leaves out automatic image settings", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.still(t, "rpicam-still")
		options := rpiOptions
		options.WhiteBalance = camera.WhiteBalanceAuto
		options.AutofocusMode = camera.AutofocusModeDefault
		options.Denoise = camera.DenoiseAuto
		cam, err := camera.New(zerolog.Nop(), options)
		require.NoError(t, err)

		_, err = cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "--nopreview --immediate --encoding jpg --thumb none --width 4056 --height 3040 -o -", tools.args(t, "rpicam-still"))
	})

	exposureTests := []struct {
		name     string
		time     time.Duration
		mode     camera.ExposureMode
		wantArgs string
	}{
		{
			name:     "passes the short profile of the automatic exposure",
			mode:     camera.ExposureModeShort,
			wantArgs: "--exposure sport ",
		},
		{
			name:     "leaves out the normal profile as the default of the tools",
			mode:     camera.ExposureModeNormal,
			wantArgs: "",
		},
		{
			name:     "leaves out the profile for a fixed exposure time",
			time:     8 * time.Millisecond,
			mode:     camera.ExposureModeShort,
			wantArgs: "--shutter 8000 ",
		},
	}

	for _, tt := range exposureTests {
		t.Run(tt.name, func(t *testing.T) {
			tools := newFakeTools(t)
			tools.preview(t, "rpicam-vid", 1)
			tools.still(t, "rpicam-still")
			options := rpiOptions
			options.ExposureTime = tt.time
			options.ExposureMode = tt.mode
			cam, err := camera.New(zerolog.Nop(), options)
			require.NoError(t, err)

			_, err = cam.Capture(t.Context())

			require.NoError(t, err)
			assert.Equal(
				t,
				"--nopreview --immediate --encoding jpg --thumb none --width 4056 --height 3040 "+tt.wantArgs+"-o -",
				tools.args(t, "rpicam-still"),
			)
		})
	}

	t.Run("keeps configured libcamera log levels", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.still(t, "rpicam-still")
		t.Setenv("LIBCAMERA_LOG_LEVELS", "*:DEBUG")
		cam, _ := newRPiCam(t)

		_, err := cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "*:DEBUG", tools.env(t, "rpicam-still"))
	})

	t.Run("logs warnings", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.still(t, "rpicam-still")
		tools.flag(t, "rpicam-still.warn")
		cam, logs := newRPiCam(t)

		_, err := cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Contains(t, logs.String(), "low light")
		assert.Contains(t, logs.String(), "rpicam-still reported warnings")
	})

	t.Run("logs progress output as debug", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.still(t, "rpicam-still")
		tools.flag(t, "rpicam-still.info")
		cam, logs := newRPiCam(t)

		_, err := cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Contains(t, logs.String(), "camera started")
		assert.NotContains(t, logs.String(), `"level":"warn"`)
	})

	t.Run("stops the running preview", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 0)
		tools.still(t, "rpicam-still")
		cam, logs := newRPiCam(t)
		out := &syncBuffer{}
		streamErr := make(chan error, 1)
		go func() {
			streamErr <- cam.StreamPreview(t.Context(), out, "frame")
		}()
		require.Eventually(t, func() bool { return strings.Contains(out.String(), "--frame") }, 5*time.Second, 10*time.Millisecond)

		data, err := cam.Capture(t.Context())

		require.NoError(t, err)
		assert.Equal(t, still, string(*data))
		select {
		case err := <-streamErr:
			assert.ErrorIs(t, err, camera.ErrPreviewStopped, "the preview reports that the capture took over the device")
		case <-time.After(5 * time.Second):
			t.Fatal("the preview kept running after the capture")
		}
		assert.NotContains(t, logs.String(), `"level":"warn"`, "an intentional stop is no warning")
	})

	t.Run("stops a hanging tool", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.hangingStill(t, "rpicam-still")
		cam, _ := newRPiCam(t)
		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()
		started := time.Now()

		data, err := cam.Capture(ctx)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Nil(t, data)
		assert.Less(t, time.Since(started), 2*time.Second, "the tool exits on SIGTERM before it has to be killed")
	})

	t.Run("returns the error of a failing tool", func(t *testing.T) {
		tools := newFakeTools(t)
		tools.preview(t, "rpicam-vid", 1)
		tools.still(t, "rpicam-still")
		tools.flag(t, "rpicam-still.fail")
		cam, _ := newRPiCam(t)

		data, err := cam.Capture(t.Context())

		require.Error(t, err)
		assert.ErrorContains(t, err, "rpicam-still failed")
		assert.ErrorContains(t, err, "device busy", "the tool's stderr explains the failure")
		assert.Nil(t, data)
	})
}

// newRPiCam builds a Raspberry Pi camera that logs to the returned buffer.
func newRPiCam(t *testing.T) (camera.Camera, *syncBuffer) {
	t.Helper()

	logger, logs := newLogger()
	cam, err := camera.New(logger, rpiOptions)
	require.NoError(t, err)
	return cam, logs
}

// failingClient fails every write, as a disconnected client does.
type failingClient struct{}

// Write fails.
func (failingClient) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}
