package config

import (
	"path/filepath"
	"strings"

	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/config/structconf"
)

// CameraConfig holds the camera settings.
type CameraConfig struct {
	Source        camera.Source `mapstructure:"source" json:"source" default:"webcam" values:"webcam,rpicam,gphoto2" usage:"Active camera source"`
	CaptureWidth  int           `mapstructure:"capture_width" json:"captureWidth" default:"4000" usage:"Still capture width in pixels of the rpicam source"`
	CaptureHeight int           `mapstructure:"capture_height" json:"captureHeight" default:"2400" usage:"Still capture height in pixels of the rpicam source"`
	PreviewWidth  int           `mapstructure:"preview_width" json:"previewWidth" default:"800" usage:"Live preview width in pixels of the rpicam source"`
	PreviewHeight int           `mapstructure:"preview_height" json:"previewHeight" default:"480" usage:"Live preview height in pixels of the rpicam source"`
	PreviewFPS    int           `mapstructure:"preview_fps" json:"previewFps" default:"25" usage:"Live preview framerate of the rpicam source"`
	OverlayPath   string        `mapstructure:"overlay_path" json:"overlayPath" usage:"Storage path of a PNG overlay composited onto every stored capture, such as overlays/frame.png, empty for none"`

	// The image settings of the rpicam source depend on the lighting of the venue and the mounting of the camera.
	ExposureTime         camera.ExposureTime  `mapstructure:"exposure_time" json:"exposureTime" default:"auto" values:"auto,1/15,1/20,1/30,1/40,1/50,1/60,1/80,1/100,1/125,1/160,1/200,1/250,1/320,1/400,1/500" usage:"Exposure time of the rpicam source with an automatic gain, such as 1/125, auto for a fully automatic exposure"`
	ExposureMode         camera.ExposureMode  `mapstructure:"exposure_mode" json:"exposureMode" default:"normal" values:"normal,short" usage:"Profile of the automatic exposure of the rpicam source, short prefers short exposure times over a low gain"`
	AnalogueGain         float64              `mapstructure:"analogue_gain" json:"analogueGain" default:"0" usage:"Fixed analogue gain of the rpicam source, 0 for an automatic gain"` //nolint:misspell // The setting keeps the term of the AnalogueGain control of libcamera.
	ExposureCompensation float64              `mapstructure:"exposure_compensation" json:"exposureCompensation" default:"0" usage:"Exposure compensation in stops of the rpicam source"`
	WhiteBalance         camera.WhiteBalance  `mapstructure:"white_balance" json:"whiteBalance" default:"auto" values:"auto,incandescent,tungsten,fluorescent,indoor,daylight,cloudy" usage:"White balance preset of the rpicam source"`
	HorizontalFlip       bool                 `mapstructure:"horizontal_flip" json:"horizontalFlip" usage:"Mirror the image of the rpicam source horizontally"`
	VerticalFlip         bool                 `mapstructure:"vertical_flip" json:"verticalFlip" usage:"Mirror the image of the rpicam source vertically"`
	Rotation             int                  `mapstructure:"rotation" json:"rotation" default:"0" values:"0,180" usage:"Image rotation in degrees of the rpicam source"`
	JPEGQuality          int                  `mapstructure:"jpeg_quality" json:"jpegQuality" default:"93" usage:"JPEG quality from 1 to 100 of the stills of the rpicam source"`
	AutofocusMode        camera.AutofocusMode `mapstructure:"autofocus_mode" json:"autofocusMode" default:"default" values:"default,manual,auto,continuous" usage:"Autofocus mode of a rpicam camera with a focus motor, such as the Camera Module 3"`
	Denoise              camera.Denoise       `mapstructure:"denoise" json:"denoise" default:"auto" values:"auto,off,cdn_off,cdn_fast,cdn_hq" usage:"Noise reduction of the rpicam source"`
}

// validateCamera normalizes and checks the camera settings.
func validateCamera(cfg *CameraConfig) error {
	var p structconf.Problems

	const (
		// maxImageDimension bounds a still capture's width or height in pixels
		maxImageDimension = 10000
		// maxPreviewDimension bounds the live preview stream's width or height in pixels
		maxPreviewDimension = 4000
		// maxPreviewFPS bounds the live preview stream's framerate
		maxPreviewFPS = 120
		// maxAnalogueGain bounds the fixed sensor gain, above the maximum of every Raspberry Pi camera sensor
		maxAnalogueGain = 32
		// maxExposureCompensation bounds the exposure compensation in stops in both directions
		maxExposureCompensation = 10
	)

	if cfg.Source == camera.SourceRPiCam {
		if cfg.CaptureWidth <= 0 || cfg.CaptureHeight <= 0 {
			p.Addf("camera.capture_width and camera.capture_height must be positive for source %q", cfg.Source)
		}
		if cfg.CaptureWidth > maxImageDimension || cfg.CaptureHeight > maxImageDimension {
			p.Addf("camera.capture_width and camera.capture_height must be at most %d", maxImageDimension)
		}
		if cfg.PreviewWidth <= 0 || cfg.PreviewHeight <= 0 {
			p.Addf("camera.preview_width and camera.preview_height must be positive for source %q", cfg.Source)
		}
		if cfg.PreviewWidth > maxPreviewDimension || cfg.PreviewHeight > maxPreviewDimension {
			p.Addf("camera.preview_width and camera.preview_height must be at most %d", maxPreviewDimension)
		}
		if cfg.PreviewFPS <= 0 {
			p.Addf("camera.preview_fps must be positive for source %q", cfg.Source)
		}
		if cfg.PreviewFPS > maxPreviewFPS {
			p.Addf("camera.preview_fps must be at most %d", maxPreviewFPS)
		}
		if cfg.CaptureWidth*cfg.PreviewHeight != cfg.PreviewWidth*cfg.CaptureHeight {
			p.Addf("camera.preview_width/camera.preview_height must have the same aspect ratio as camera.capture_width/camera.capture_height")
		}
		// A gain below 1 would darken the image, which the sensor cannot do
		if cfg.AnalogueGain != 0 && (cfg.AnalogueGain < 1 || cfg.AnalogueGain > maxAnalogueGain) {
			p.Addf("camera.analogue_gain must be 0 or between 1 and %d", maxAnalogueGain)
		}
		if cfg.ExposureCompensation < -maxExposureCompensation || cfg.ExposureCompensation > maxExposureCompensation {
			p.Addf("camera.exposure_compensation must be between %d and %d", -maxExposureCompensation, maxExposureCompensation)
		}
		if cfg.JPEGQuality < 1 || cfg.JPEGQuality > 100 {
			p.Addf("camera.jpeg_quality must be between 1 and 100")
		}
	}

	cfg.OverlayPath = strings.TrimSpace(cfg.OverlayPath)
	if cfg.OverlayPath != "" && !strings.EqualFold(filepath.Ext(cfg.OverlayPath), ".png") {
		p.Addf("camera.overlay_path must point to a .png file, got %q", cfg.OverlayPath)
	}
	if len(cfg.OverlayPath) > maxPath {
		p.Addf("camera.overlay_path must be at most %d characters", maxPath)
	}

	return p.Err()
}
