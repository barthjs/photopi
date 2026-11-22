// Package camera provides a Camera interface for streaming live preview and taking stills
// behind a driver-agnostic interface, with interchangeable webcam, rpicam, and gphoto2 backends.
package camera

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Source selects the camera backend.
type Source string

const (
	SourceWebcam Source = "webcam"
	SourceRPiCam Source = "rpicam"
	SourceGphoto Source = "gphoto2"
)

// ExposureTime is the exposure time of the rpicam source as a fraction of a second,
// written like on a camera, or ExposureTimeAuto for a fully automatic exposure.
type ExposureTime string

// ExposureTimeAuto lets the camera choose the exposure time and the gain.
const ExposureTimeAuto ExposureTime = "auto"

// Duration returns the exposure time, 0 for ExposureTimeAuto or a value that is no fraction of a second.
func (e ExposureTime) Duration() time.Duration {
	numerator, denominator, found := strings.Cut(string(e), "/")
	if !found {
		return 0
	}

	top, err := strconv.Atoi(numerator)
	if err != nil || top <= 0 {
		return 0
	}
	bottom, err := strconv.Atoi(denominator)
	if err != nil || bottom <= 0 {
		return 0
	}

	return time.Duration(top) * time.Second / time.Duration(bottom)
}

// ExposureMode selects the profile the automatic exposure of the rpicam source follows.
type ExposureMode string

const (
	// ExposureModeNormal lengthens the exposure time first and raises the gain only once the time is long.
	ExposureModeNormal ExposureMode = "normal"
	// ExposureModeShort keeps the exposure time short and raises the gain earlier,
	// so moving guests stay sharper at the cost of more noise.
	ExposureModeShort ExposureMode = "short"
)

// WhiteBalance selects the white balance preset of the rpicam source.
type WhiteBalance string

const (
	WhiteBalanceAuto         WhiteBalance = "auto"
	WhiteBalanceIncandescent WhiteBalance = "incandescent"
	WhiteBalanceTungsten     WhiteBalance = "tungsten"
	WhiteBalanceFluorescent  WhiteBalance = "fluorescent"
	WhiteBalanceIndoor       WhiteBalance = "indoor"
	WhiteBalanceDaylight     WhiteBalance = "daylight"
	WhiteBalanceCloudy       WhiteBalance = "cloudy"
)

// AutofocusMode selects the autofocus mode of a rpicam camera with a focus motor.
type AutofocusMode string

const (
	// AutofocusModeDefault keeps the mode the camera starts with and is the only mode for a lens without a motor.
	AutofocusModeDefault    AutofocusMode = "default"
	AutofocusModeManual     AutofocusMode = "manual"
	AutofocusModeAuto       AutofocusMode = "auto"
	AutofocusModeContinuous AutofocusMode = "continuous"
)

// Denoise selects the noise reduction of the rpicam source.
type Denoise string

const (
	// DenoiseAuto lets the tools choose, which is a fast reduction for the preview and a thorough one for the still.
	DenoiseAuto    Denoise = "auto"
	DenoiseOff     Denoise = "off"
	DenoiseCDNOff  Denoise = "cdn_off"
	DenoiseCDNFast Denoise = "cdn_fast"
	DenoiseCDNHQ   Denoise = "cdn_hq"
)

var (
	// ErrUnsupported is returned when a feature is not available with the current source.
	ErrUnsupported = errors.New("camera: operation not supported by this source")
	// ErrPreviewStopped ends a preview stream that a capture stopped to take the photo with the same device.
	ErrPreviewStopped = errors.New("camera: preview stopped for a capture")
)

// Camera streams a live preview and takes stills for a backend-driven camera source.
type Camera interface {
	// StreamPreview writes a continuous MJPEG multipart/x-mixed-replace body to w,
	// using the given boundary, until ctx is canceled or the underlying source stops producing frames.
	StreamPreview(ctx context.Context, w io.Writer, boundary string) error
	// Capture takes one full-resolution still and returns its encoded JPEG bytes.
	Capture(ctx context.Context) (*[]byte, error)
}

// Options hold the settings required to construct a Camera instance.
// The image settings apply to the rpicam source only, where their zero values keep the tools' own behavior.
type Options struct {
	Source        Source
	CaptureWidth  int
	CaptureHeight int
	PreviewWidth  int
	PreviewHeight int
	PreviewFPS    int
	// ExposureTime fixes the exposure time while the gain stays automatic, 0 for a fully automatic exposure.
	ExposureTime time.Duration
	// ExposureMode selects the profile of the automatic exposure, it has no effect with a fixed ExposureTime.
	ExposureMode ExposureMode
	// AnalogueGain fixes the sensor gain, 0 for an automatic gain.
	AnalogueGain float64
	// ExposureCompensation shifts the automatic exposure in stops.
	ExposureCompensation float64
	WhiteBalance         WhiteBalance
	HorizontalFlip       bool
	VerticalFlip         bool
	// Rotation turns the image by 0 or 180 degrees, the only rotations the sensor supports.
	Rotation int
	// JPEGQuality is the JPEG quality of the still from 1 to 100, 0 for the tool's default.
	JPEGQuality   int
	AutofocusMode AutofocusMode
	Denoise       Denoise
}

// New creates a new Camera instance for the configured Source.
func New(logger zerolog.Logger, options Options) (Camera, error) {
	switch options.Source {
	case SourceWebcam:
		return newWebcamCamera(), nil
	case SourceRPiCam:
		return newRPiCam(logger, options)
	case SourceGphoto:
		return newGphotoCam(logger)
	default:
		return nil, fmt.Errorf("camera: unknown source %q", options.Source)
	}
}
