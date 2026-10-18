package camera

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/rs/zerolog"
)

// libcameraLogLevels limits the libcamera log output of the rpicam tools to warnings and errors,
// which otherwise fills stderr with informational messages on every start.
const libcameraLogLevels = "LIBCAMERA_LOG_LEVELS=*:WARN"

// rpiCam drives the Raspberry Pi camera through the rpicam-apps command-line tools. rpicam-vid feeds the live preview,
// and rpicam-still takes the full-resolution still. The ISP processes the sensor data in hardware,
// while the JPEG encoding of the preview frames and the still runs on the CPU, since the Raspberry Pi 5
// has no hardware JPEG encoder. The Go process only starts the tools and forwards their output.
//
// libcamera only ever lets one process hold the camera device at a time:
// a second one fails immediately with "Device or resource busy". device coordinates that:
// since the live preview (StreamPreview) runs for as long as a client stays connected,
// a still (Capture) requested while it is running has to actively stop it
// and wait for rpicam-vid to actually release the device first.
type rpiCam struct {
	cfg    Options
	logger zerolog.Logger
	env    []string

	device *deviceSession
}

// newRPiCam creates a new rpiCam instance and verifies the rpicam tools are on PATH.
func newRPiCam(logger zerolog.Logger, options Options) (*rpiCam, error) {
	for _, bin := range []string{"rpicam-vid", "rpicam-still"} {
		if _, err := exec.LookPath(bin); err != nil {
			return nil, fmt.Errorf("camera: required tool %q not found in PATH: %w", bin, err)
		}
	}

	return &rpiCam{
		cfg:    options,
		logger: logger,
		env:    rpicamEnv(),
		device: newDeviceSession(),
	}, nil
}

// rpicamEnv returns the environment of the rpicam tools, which limits libcamera to warnings
// unless the environment of this process already sets the log levels, for example to debug the camera.
func rpicamEnv() []string {
	env := os.Environ()
	if _, ok := os.LookupEnv("LIBCAMERA_LOG_LEVELS"); ok {
		return env
	}
	return append(env, libcameraLogLevels)
}

// StreamPreview starts rpicam-vid at the configured preview resolution and framerate
// and forwards its raw MJPEG output to w as a multipart/x-mixed-replace stream, until ctx is canceled,
// the client disconnects, or a concurrent Capture stops it to take the shot over the same camera device.
func (c *rpiCam) StreamPreview(ctx context.Context, w io.Writer, boundary string) error {
	return streamPreviewProcess(ctx, c.logger, c.device, c.previewCommand(), w, boundary)
}

// previewCommand returns the rpicam-vid invocation of the live preview.
func (c *rpiCam) previewCommand() command {
	args := []string{
		"--nopreview",
		"--codec", "mjpeg",
		"--width", strconv.Itoa(c.cfg.PreviewWidth),
		"--height", strconv.Itoa(c.cfg.PreviewHeight),
		"--framerate", strconv.Itoa(c.cfg.PreviewFPS),
		"--timeout", "0",
		// Hands every frame to the pipe at once instead of when the output buffer is full
		"--flush",
	}
	args = append(args, c.imageArgs()...)
	args = append(args, "-o", "-")

	return command{
		label: "rpicam-vid",
		name:  "rpicam-vid",
		args:  args,
		env:   c.env,
	}
}

// Capture runs rpicam-still once and returns the resulting JPEG bytes at the configured capture resolution.
// If a live preview stream is currently running, it is stopped first,
// and Capture waits for rpicam-vid to fully release the camera device before taking the shot.
func (c *rpiCam) Capture(ctx context.Context) (*[]byte, error) {
	return captureProcess(ctx, c.logger, c.device, c.stillCommand())
}

// stillCommand returns the rpicam-still invocation of a capture.
func (c *rpiCam) stillCommand() command {
	args := []string{
		"--nopreview",
		"--immediate",
		"--encoding", "jpg",
		// The EXIF thumbnail would show the photo without the overlay the backend draws over it
		"--thumb", "none",
		"--width", strconv.Itoa(c.cfg.CaptureWidth),
		"--height", strconv.Itoa(c.cfg.CaptureHeight),
	}
	if c.cfg.JPEGQuality > 0 {
		args = append(args, "--quality", strconv.Itoa(c.cfg.JPEGQuality))
	}
	args = append(args, c.imageArgs()...)
	args = append(args, "-o", "-")

	return command{
		label: "rpicam-still",
		name:  "rpicam-still",
		args:  args,
		env:   c.env,
	}
}

// imageArgs returns the image settings shared by the preview and the still, so the preview looks like the photo.
// Settings at their automatic value are left out, so the tools apply their own defaults.
func (c *rpiCam) imageArgs() []string {
	var args []string
	if c.cfg.ExposureTime > 0 {
		args = append(args, "--shutter", strconv.FormatInt(c.cfg.ExposureTime.Microseconds(), 10))
	}
	// rpicam calls the profile with short exposure times sport
	if c.cfg.ExposureTime == 0 && c.cfg.ExposureMode == ExposureModeShort {
		args = append(args, "--exposure", "sport")
	}
	if c.cfg.AnalogueGain > 0 {
		args = append(args, "--gain", strconv.FormatFloat(c.cfg.AnalogueGain, 'f', -1, 64))
	}
	if c.cfg.ExposureCompensation != 0 {
		args = append(args, "--ev", strconv.FormatFloat(c.cfg.ExposureCompensation, 'f', -1, 64))
	}
	if c.cfg.WhiteBalance != "" && c.cfg.WhiteBalance != WhiteBalanceAuto {
		args = append(args, "--awb", string(c.cfg.WhiteBalance))
	}
	if c.cfg.HorizontalFlip {
		args = append(args, "--hflip")
	}
	if c.cfg.VerticalFlip {
		args = append(args, "--vflip")
	}
	if c.cfg.Rotation != 0 {
		args = append(args, "--rotation", strconv.Itoa(c.cfg.Rotation))
	}
	if c.cfg.Denoise != "" && c.cfg.Denoise != DenoiseAuto {
		args = append(args, "--denoise", string(c.cfg.Denoise))
	}
	// Only passed when set, so a camera without a focus motor never receives the option
	if c.cfg.AutofocusMode != "" && c.cfg.AutofocusMode != AutofocusModeDefault {
		args = append(args, "--autofocus-mode", string(c.cfg.AutofocusMode))
	}
	return args
}
