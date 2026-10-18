package app

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/service"
	"github.com/barthjs/photopi/internal/share"
)

func TestStorageOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the s3 settings", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.StorageConfig{
			Driver: media.DriverS3,
			S3: config.S3Config{
				Endpoint:     f.URL(),
				Region:       f.LetterN(8),
				Bucket:       f.LetterN(8),
				AccessKey:    f.LetterN(16),
				SecretKey:    f.LetterN(32),
				UsePathStyle: true,
			},
		}

		got, err := StorageOptions(cfg)

		require.NoError(t, err)
		assert.Equal(t, media.Options{
			Driver:         media.DriverS3,
			S3Endpoint:     cfg.S3.Endpoint,
			S3Region:       cfg.S3.Region,
			S3Bucket:       cfg.S3.Bucket,
			S3AccessKey:    cfg.S3.AccessKey,
			S3SecretKey:    cfg.S3.SecretKey,
			S3UsePathStyle: cfg.S3.UsePathStyle,
			SignedURLBase:  api.MediaPath,
		}, got)
	})

	t.Run("keeps the configured path of the local driver", func(t *testing.T) {
		t.Parallel()

		path := t.TempDir()

		got, err := StorageOptions(config.StorageConfig{
			Driver: media.DriverLocal,
			Path:   path,
		})

		require.NoError(t, err)
		assert.Equal(t, path, got.LocalPath)
	})
}

func TestI18nOptions(t *testing.T) {
	t.Parallel()

	t.Run("reports missing translations in development", func(t *testing.T) {
		t.Parallel()

		key := apitest.Faker(t).LetterN(16)
		var logs bytes.Buffer
		cfg := config.Config{
			App: config.AppConfig{
				Env: config.EnvDevelopment,
			},
		}

		bundle, err := i18n.New(i18nOptions(zerolog.New(&logs), cfg)...)

		require.NoError(t, err)
		bundle.T("en", key, nil)
		assert.Contains(t, logs.String(), key)
	})

	t.Run("stays quiet in production", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			App: config.AppConfig{
				Env: config.EnvProduction,
			},
		}

		got := i18nOptions(zerolog.Nop(), cfg)

		assert.Empty(t, got)
	})
}

func TestMailerOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the mail settings", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.MailConfig{
			Driver:          mailer.DriverSMTP,
			Host:            f.DomainName(),
			Port:            f.IntRange(1, 65535),
			Username:        f.Username(),
			Password:        f.Password(true, true, true, false, false, 16),
			FromAddress:     f.Email(),
			FromName:        f.Name(),
			ReplyTo:         f.Email(),
			MessageIDDomain: f.DomainName(),
		}

		got := mailerOptions(cfg)

		assert.Equal(t, mailer.Options{
			Driver:          cfg.Driver,
			FromAddress:     cfg.FromAddress,
			FromName:        cfg.FromName,
			ReplyTo:         cfg.ReplyTo,
			SMTPHost:        cfg.Host,
			SMTPPort:        cfg.Port,
			SMTPUsername:    cfg.Username,
			SMTPPassword:    cfg.Password,
			MessageIDDomain: cfg.MessageIDDomain,
		}, got)
	})
}

func TestCameraOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the camera settings", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg, err := config.Defaults()
		require.NoError(t, err)
		cameraCfg := cfg.Camera
		cameraCfg.Source = camera.SourceRPiCam
		cameraCfg.CaptureWidth = f.IntRange(100, 4000)
		cameraCfg.PreviewFPS = f.IntRange(1, 60)
		cameraCfg.AnalogueGain = f.Float64Range(1, 16)
		cameraCfg.HorizontalFlip = true
		cameraCfg.Rotation = 180

		got := cameraOptions(cameraCfg)

		assert.Equal(t, camera.Options{
			Source:               cameraCfg.Source,
			CaptureWidth:         cameraCfg.CaptureWidth,
			CaptureHeight:        cameraCfg.CaptureHeight,
			PreviewWidth:         cameraCfg.PreviewWidth,
			PreviewHeight:        cameraCfg.PreviewHeight,
			PreviewFPS:           cameraCfg.PreviewFPS,
			ExposureTime:         cameraCfg.ExposureTime.Duration(),
			ExposureMode:         cameraCfg.ExposureMode,
			AnalogueGain:         cameraCfg.AnalogueGain,
			ExposureCompensation: cameraCfg.ExposureCompensation,
			WhiteBalance:         cameraCfg.WhiteBalance,
			HorizontalFlip:       cameraCfg.HorizontalFlip,
			VerticalFlip:         cameraCfg.VerticalFlip,
			Rotation:             cameraCfg.Rotation,
			JPEGQuality:          cameraCfg.JPEGQuality,
			AutofocusMode:        cameraCfg.AutofocusMode,
			Denoise:              cameraCfg.Denoise,
		}, got)
	})
}

func TestCloudProviderOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the nextcloud settings", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.CloudShareConfig{
			Driver: share.DriverNextcloud,
			Nextcloud: config.NextcloudConfig{
				URL:      f.URL(),
				Username: f.Username(),
				Password: f.LetterN(16),
				Folder:   f.LetterN(8),
			},
		}

		got := cloudProviderOptions(cfg)

		assert.Equal(t, share.Options{
			Driver:            cfg.Driver,
			NextcloudURL:      cfg.Nextcloud.URL,
			NextcloudUsername: cfg.Nextcloud.Username,
			NextcloudPassword: cfg.Nextcloud.Password,
			NextcloudFolder:   cfg.Nextcloud.Folder,
		}, got)
	})
}

func TestAuthOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the admin password", func(t *testing.T) {
		t.Parallel()

		cfg := config.AppConfig{
			Secret: apitest.Faker(t).LetterN(16),
		}

		got := authOptions(cfg)

		assert.Equal(t, service.AuthOptions{
			Password: cfg.Secret,
		}, got)
	})
}

func TestOverlayOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the path of the active overlay", func(t *testing.T) {
		t.Parallel()

		cfg := config.CameraConfig{
			OverlayPath: "overlays/" + apitest.Faker(t).LetterN(8) + ".png",
		}

		got := overlayOptions(cfg)

		assert.Equal(t, service.OverlayOptions{
			ActivePath: cfg.OverlayPath,
		}, got)
	})
}

func TestCaptureOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the capture settings with the capture size of the rpicam source", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		capture := config.CaptureConfig{
			ImagesPerRun:  f.IntRange(1, 10),
			FilePrefix:    f.LetterN(8),
			EventName:     f.LetterN(8),
			SaveOriginals: true,
		}
		cameraCfg := config.CameraConfig{
			Source:        camera.SourceRPiCam,
			CaptureWidth:  f.IntRange(100, 4000),
			CaptureHeight: f.IntRange(100, 4000),
			OverlayPath:   "overlays/" + f.LetterN(8) + ".png",
		}

		got := captureOptions(capture, cameraCfg)

		assert.Equal(t, service.CaptureOptions{
			ImagesPerRun:  capture.ImagesPerRun,
			FilePrefix:    capture.FilePrefix,
			EventName:     capture.EventName,
			SaveOriginals: capture.SaveOriginals,
			OverlayPath:   cameraCfg.OverlayPath,
			CaptureWidth:  cameraCfg.CaptureWidth,
			CaptureHeight: cameraCfg.CaptureHeight,
		}, got)
	})

	t.Run("leaves out the capture size of the other sources", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cameraCfg := config.CameraConfig{
			Source:        camera.SourceWebcam,
			CaptureWidth:  f.IntRange(100, 4000),
			CaptureHeight: f.IntRange(100, 4000),
		}

		got := captureOptions(config.CaptureConfig{}, cameraCfg)

		assert.Zero(t, got.CaptureWidth)
		assert.Zero(t, got.CaptureHeight)
	})
}

func TestSystemOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the commands", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		system := config.SystemConfig{
			ServerRestartCommand: f.LoremIpsumSentence(4),
			ServerStopCommand:    f.LoremIpsumSentence(4),
			KioskRestartCommand:  f.LoremIpsumSentence(4),
			KioskStopCommand:     f.LoremIpsumSentence(4),
			AllRestartCommand:    f.LoremIpsumSentence(4),
			AllStopCommand:       f.LoremIpsumSentence(4),
		}

		got := systemOptions(system)

		assert.Equal(t, service.SystemOptions{
			ServerRestartCommand: system.ServerRestartCommand,
			ServerStopCommand:    system.ServerStopCommand,
			KioskRestartCommand:  system.KioskRestartCommand,
			KioskStopCommand:     system.KioskStopCommand,
			AllRestartCommand:    system.AllRestartCommand,
			AllStopCommand:       system.AllStopCommand,
		}, got)
	})
}

func TestPrintOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the print settings with the event name and file prefix of the captures", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		printCfg := config.PrintConfig{
			Enabled:        true,
			PrinterName:    f.LetterN(8),
			LayoutID:       f.LetterN(8),
			SubmissionMode: string(service.PrintSubmissionRaw),
		}
		capture := config.CaptureConfig{
			FilePrefix: f.LetterN(8),
			EventName:  f.LetterN(8),
		}

		got := printOptions(printCfg, capture)

		assert.Equal(t, service.PrintOptions{
			Enabled:        printCfg.Enabled,
			PrinterName:    printCfg.PrinterName,
			LayoutID:       printCfg.LayoutID,
			SubmissionMode: service.PrintSubmissionRaw,
			EventName:      capture.EventName,
			FilePrefix:     capture.FilePrefix,
		}, got)
	})
}

func TestShareOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the share settings with the base url of the self share", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.Config{
			App: config.AppConfig{
				Name:     f.LetterN(8),
				Language: config.Language("de"),
			},
			Capture: config.CaptureConfig{
				EventName: f.LetterN(8),
			},
			Print: config.PrintConfig{
				Enabled: true,
			},
			Share: config.ShareConfig{
				Email: config.EmailShareConfig{
					Enabled: true,
					Subject: f.LoremIpsumSentence(4),
				},
				Cloud: config.CloudShareConfig{
					Driver: share.DriverSelf,
					Self: config.SelfShareConfig{
						BaseURL: f.URL(),
					},
				},
			},
		}
		signingKey := []byte(f.LetterN(32))

		got := shareOptions(cfg, signingKey)

		assert.Equal(t, service.ShareOptions{
			AppName:        cfg.App.Name,
			Language:       string(cfg.App.Language),
			EventName:      cfg.Capture.EventName,
			EmailEnabled:   cfg.Share.Email.Enabled,
			EmailSubject:   cfg.Share.Email.Subject,
			PrintEnabled:   cfg.Print.Enabled,
			SelfBaseURL:    cfg.Share.Cloud.Self.BaseURL,
			SelfSigningKey: signingKey,
		}, got)
	})

	t.Run("leaves out the base url of the other drivers", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.Config{
			Share: config.ShareConfig{
				Cloud: config.CloudShareConfig{
					Driver: share.DriverNextcloud,
					Self: config.SelfShareConfig{
						BaseURL: f.URL(),
					},
				},
			},
		}

		got := shareOptions(cfg, []byte(f.LetterN(32)))

		assert.Empty(t, got.SelfBaseURL)
	})
}

func TestAPIOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the settings the api reads", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg, err := config.Defaults()
		require.NoError(t, err)
		cfg.App.Name = f.LetterN(8)
		cfg.App.Language = config.LanguageDE
		cfg.Theme.Primary = f.Color()
		cfg.Camera.OverlayPath = "overlays/" + f.LetterN(8) + ".png"
		cfg.Capture.WelcomeMessage = f.LoremIpsumSentence(4)
		cfg.Capture.AdminEmail = f.Email()
		cfg.Server.CORSAllowedOrigins = []string{
			f.URL(),
		}
		cfg.Server.CORSAllowCredentials = true

		got := apiOptions(cfg)

		assert.Equal(t, api.Options{
			Name:     cfg.App.Name,
			Language: string(cfg.App.Language),
			Theme: api.ThemeSettings{
				Primary:   cfg.Theme.Primary,
				Secondary: cfg.Theme.Secondary,
				Success:   cfg.Theme.Success,
				Info:      cfg.Theme.Info,
				Warning:   cfg.Theme.Warning,
				Error:     cfg.Theme.Error,
				Neutral:   cfg.Theme.Neutral,
			},
			Camera: api.CameraOptions{
				Source:        cfg.Camera.Source,
				CaptureWidth:  cfg.Camera.CaptureWidth,
				CaptureHeight: cfg.Camera.CaptureHeight,
				PreviewWidth:  cfg.Camera.PreviewWidth,
				PreviewHeight: cfg.Camera.PreviewHeight,
				PreviewFPS:    cfg.Camera.PreviewFPS,
				OverlayPath:   cfg.Camera.OverlayPath,
			},
			Capture: api.CaptureOptions{
				ImagesPerRun:   cfg.Capture.ImagesPerRun,
				WelcomeMessage: cfg.Capture.WelcomeMessage,
				AdminEmail:     cfg.Capture.AdminEmail,
			},
			TrustedProxies:       cfg.Server.TrustedProxies,
			CORSAllowedOrigins:   cfg.Server.CORSAllowedOrigins,
			CORSAllowCredentials: cfg.Server.CORSAllowCredentials,
			Docs:                 true,
		}, got)
	})

}
