package config

import (
	"strings"

	"github.com/barthjs/photopi/internal/config/structconf"
)

// CaptureConfig holds the capture settings.
type CaptureConfig struct {
	ImagesPerRun   int    `mapstructure:"images_per_run" json:"imagesPerRun" default:"4" usage:"Number of captures per run"`
	FilePrefix     string `mapstructure:"file_prefix" json:"filePrefix" default:"PhotoPi" usage:"Filename prefix of stored captures"`
	EventName      string `mapstructure:"event_name" json:"eventName" default:"PhotoPi" usage:"Event name in share emails, also the storage folder of the captures"`
	WelcomeMessage string `mapstructure:"welcome_message" json:"welcomeMessage" usage:"Message of the welcome screen, empty for none"`
	AdminEmail     string `mapstructure:"admin_email" json:"adminEmail" usage:"Contact address shown to a guest when emailing their captures fails"`
	SaveOriginals  bool   `mapstructure:"save_originals" json:"saveOriginals" default:"true" usage:"Store an unmodified copy of each capture under originals/, requires camera.overlay_path"`
}

// validateCapture normalizes and checks the capture settings.
func validateCapture(cfg *CaptureConfig) error {
	var p structconf.Problems

	const (
		maxImagesPerRun   = 20
		maxWelcomeMessage = 70
	)

	if cfg.ImagesPerRun <= 0 {
		p.Addf("capture.images_per_run must be positive")
	}
	if cfg.ImagesPerRun > maxImagesPerRun {
		p.Addf("capture.images_per_run must be at most %d", maxImagesPerRun)
	}

	cfg.FilePrefix = strings.TrimSpace(cfg.FilePrefix)
	if cfg.FilePrefix == "" {
		p.Addf("capture.file_prefix must not be empty")
	}
	if len(cfg.FilePrefix) > maxShortText {
		p.Addf("capture.file_prefix must be at most %d characters", maxShortText)
	}
	if strings.ContainsAny(cfg.FilePrefix, `/\`) {
		p.Addf("capture.file_prefix must not contain path separators, got %q", cfg.FilePrefix)
	}

	cfg.EventName = strings.TrimSpace(cfg.EventName)
	if cfg.EventName == "" {
		p.Addf("capture.event_name must not be empty")
	}
	if len(cfg.EventName) > maxShortText {
		p.Addf("capture.event_name must be at most %d characters", maxShortText)
	}
	if strings.ContainsAny(cfg.EventName, `/\`) {
		p.Addf("capture.event_name must not contain path separators, got %q", cfg.EventName)
	}

	if len(cfg.WelcomeMessage) > maxWelcomeMessage {
		p.Addf("capture.welcome_message must be at most %d characters", maxWelcomeMessage)
	}

	return p.Err()
}
