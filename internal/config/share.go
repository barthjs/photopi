package config

import (
	"strings"

	"github.com/barthjs/photopi/internal/config/structconf"
	"github.com/barthjs/photopi/internal/share"
)

// EmailShareConfig holds the settings of sharing captures by email.
type EmailShareConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled" default:"true" usage:"Allow guests to email themselves their captures"`
	Subject string `mapstructure:"subject" json:"subject" usage:"Subject of the share email, empty for the built-in subject"`
}

// NextcloudConfig holds the settings of the nextcloud share driver.
type NextcloudConfig struct {
	URL      string `mapstructure:"url" json:"url" usage:"Nextcloud base URL of the nextcloud driver"`
	Username string `mapstructure:"username" json:"username" usage:"Nextcloud username of the nextcloud driver"`
	Password string `mapstructure:"password" json:"password" secret:"true" usage:"Nextcloud app password of the nextcloud driver"`
	Folder   string `mapstructure:"folder" json:"folder" usage:"Nextcloud folder the nextcloud driver uploads into"`
}

// SelfShareConfig holds the settings of the self share driver.
type SelfShareConfig struct {
	BaseURL string `mapstructure:"base_url" json:"baseUrl" usage:"Public base URL guests reach this server at, for the QR code link of the self driver, such as http://192.168.4.1:8080"`
}

// CloudShareConfig holds the settings of the cloud share.
type CloudShareConfig struct {
	Driver    share.Driver    `mapstructure:"driver" json:"driver" default:"" usage:"Cloud share driver (nextcloud, self), empty disables the cloud share"`
	Nextcloud NextcloudConfig `mapstructure:"nextcloud" json:"nextcloud"`
	Self      SelfShareConfig `mapstructure:"self" json:"self"`
}

// ShareConfig holds the share settings.
type ShareConfig struct {
	Email EmailShareConfig `mapstructure:"email" json:"email"`
	Cloud CloudShareConfig `mapstructure:"cloud" json:"cloud"`
}

// validateShare normalizes and checks the share settings.
// The cloud driver has no values tag, since its empty value disables the cloud share.
func validateShare(cfg *ShareConfig) error {
	var p structconf.Problems

	cfg.Email.Subject = strings.TrimSpace(cfg.Email.Subject)
	if len(cfg.Email.Subject) > maxText {
		p.Addf("share.email.subject must be at most %d characters", maxText)
	}

	switch cfg.Cloud.Driver {
	case share.DriverNone:
	case share.DriverNextcloud:
		cfg.Cloud.Nextcloud.URL = strings.TrimRight(cfg.Cloud.Nextcloud.URL, "/")
		cfg.Cloud.Nextcloud.Folder = strings.Trim(cfg.Cloud.Nextcloud.Folder, "/")
		if cfg.Cloud.Nextcloud.URL == "" || cfg.Cloud.Nextcloud.Username == "" || cfg.Cloud.Nextcloud.Password == "" {
			p.Addf("share driver nextcloud requires SHARE_CLOUD_NEXTCLOUD_URL, SHARE_CLOUD_NEXTCLOUD_USERNAME and SHARE_CLOUD_NEXTCLOUD_PASSWORD")
		}
		if len(cfg.Cloud.Nextcloud.Folder) > maxPath {
			p.Addf("share.cloud.nextcloud.folder must be at most %d characters", maxPath)
		}
	case share.DriverSelf:
		cfg.Cloud.Self.BaseURL = strings.TrimRight(cfg.Cloud.Self.BaseURL, "/")
		if cfg.Cloud.Self.BaseURL == "" {
			p.Addf("share driver self requires SHARE_CLOUD_SELF_BASE_URL")
		}
	default:
		p.Addf("share.cloud.driver must be %q, %q, or %q, got %q", share.DriverNone, share.DriverNextcloud, share.DriverSelf, cfg.Cloud.Driver)
	}

	return p.Err()
}
