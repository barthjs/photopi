package app

import (
	"github.com/rs/zerolog"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/service"
	"github.com/barthjs/photopi/internal/share"
)

// splashColor is the background of the app icon, which the splash screen of the installed app shows around it.
const splashColor = "#115e59"

// apiOptions maps the configuration onto the options of the API.
func apiOptions(cfg config.Config) api.Options {
	return api.Options{
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
		SplashColor: splashColor,
		Camera: api.CameraOptions{
			Source:      cfg.Camera.Source,
			OverlayPath: cfg.Camera.OverlayPath,
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
	}
}

// authOptions maps the app settings onto the options of the auth service.
func authOptions(cfg config.AppConfig) service.AuthOptions {
	return service.AuthOptions{
		Password: cfg.Secret,
	}
}

// cameraOptions maps the camera settings onto the options of the camera package.
func cameraOptions(cfg config.CameraConfig) camera.Options {
	return camera.Options{
		Source:               cfg.Source,
		CaptureWidth:         cfg.CaptureWidth,
		CaptureHeight:        cfg.CaptureHeight,
		PreviewWidth:         cfg.PreviewWidth,
		PreviewHeight:        cfg.PreviewHeight,
		PreviewFPS:           cfg.PreviewFPS,
		ExposureTime:         cfg.ExposureTime.Duration(),
		ExposureMode:         cfg.ExposureMode,
		AnalogueGain:         cfg.AnalogueGain,
		ExposureCompensation: cfg.ExposureCompensation,
		WhiteBalance:         cfg.WhiteBalance,
		HorizontalFlip:       cfg.HorizontalFlip,
		VerticalFlip:         cfg.VerticalFlip,
		Rotation:             cfg.Rotation,
		JPEGQuality:          cfg.JPEGQuality,
		AutofocusMode:        cfg.AutofocusMode,
		Denoise:              cfg.Denoise,
	}
}

// captureOptions maps the capture settings and the camera onto the capture service options.
// Only the rpicam source has a configured capture size, the others deliver whatever size the camera takes.
func captureOptions(cfg config.CaptureConfig, cameraCfg config.CameraConfig) service.CaptureOptions {
	options := service.CaptureOptions{
		ImagesPerRun:  cfg.ImagesPerRun,
		FilePrefix:    cfg.FilePrefix,
		EventName:     cfg.EventName,
		SaveOriginals: cfg.SaveOriginals,
		OverlayPath:   cameraCfg.OverlayPath,
	}
	if cameraCfg.Source == camera.SourceRPiCam {
		options.CaptureWidth = cameraCfg.CaptureWidth
		options.CaptureHeight = cameraCfg.CaptureHeight
	}

	return options
}

// cloudProviderOptions maps the cloud share settings onto the options of the share package.
func cloudProviderOptions(cfg config.CloudShareConfig) share.Options {
	return share.Options{
		Driver:            cfg.Driver,
		NextcloudURL:      cfg.Nextcloud.URL,
		NextcloudUsername: cfg.Nextcloud.Username,
		NextcloudPassword: cfg.Nextcloud.Password,
		NextcloudFolder:   cfg.Nextcloud.Folder,
	}
}

// i18nOptions reports missing translations as warnings during development,
// where a developer can still add them, and stays quiet in production.
func i18nOptions(logger zerolog.Logger, cfg config.Config) []i18n.Option {
	if cfg.App.Env != config.EnvDevelopment {
		return nil
	}
	return []i18n.Option{i18n.WithMissingKeyHandler(func(locale, key string) {
		logger.Warn().Str("locale", locale).Str("key", key).Msg("missing translation")
	})}
}

// mailerOptions maps the mail settings onto the options of the mailer.
func mailerOptions(cfg config.MailConfig) mailer.Options {
	return mailer.Options{
		Driver:       cfg.Driver,
		FromAddress:  cfg.FromAddress,
		FromName:     cfg.FromName,
		ReplyTo:      cfg.ReplyTo,
		SMTPHost:     cfg.Host,
		SMTPPort:     cfg.Port,
		SMTPUsername: cfg.Username,
		SMTPPassword: cfg.Password,

		MessageIDDomain: cfg.MessageIDDomain,
	}
}

// overlayOptions maps the camera settings onto the options of the overlay service.
func overlayOptions(cfg config.CameraConfig) service.OverlayOptions {
	return service.OverlayOptions{
		ActivePath: cfg.OverlayPath,
	}
}

// printOptions maps the print settings and the event name and file prefix of the captures onto the print service options.
func printOptions(cfg config.PrintConfig, capture config.CaptureConfig) service.PrintOptions {
	return service.PrintOptions{
		Enabled:        cfg.Enabled,
		PrinterName:    cfg.PrinterName,
		Layout:         cfg.Layout,
		SubmissionMode: service.PrintSubmissionMode(cfg.SubmissionMode),
		EventName:      capture.EventName,
		FilePrefix:     capture.FilePrefix,
	}
}

// shareOptions maps the configuration onto the share service options.
// The self share driver shares through the public share page at its base URL, signed with selfSigningKey.
func shareOptions(cfg config.Config, selfSigningKey []byte) service.ShareOptions {
	selfBaseURL := ""
	if cfg.Share.Cloud.Driver == share.DriverSelf {
		selfBaseURL = cfg.Share.Cloud.Self.BaseURL
	}

	return service.ShareOptions{
		AppName:      cfg.App.Name,
		Language:     string(cfg.App.Language),
		EventName:    cfg.Capture.EventName,
		EmailEnabled: cfg.Share.Email.Enabled,
		EmailSubject: cfg.Share.Email.Subject,
		PrintEnabled: cfg.Print.Enabled,

		SelfBaseURL:    selfBaseURL,
		SelfSigningKey: selfSigningKey,
	}
}

// StorageOptions maps the storage settings onto the options of the media package.
// An empty path of the local driver selects the application's data directory.
func StorageOptions(cfg config.StorageConfig) (media.Options, error) {
	options := media.Options{
		Driver:         cfg.Driver,
		LocalPath:      cfg.Path,
		S3Endpoint:     cfg.S3.Endpoint,
		S3Region:       cfg.S3.Region,
		S3Bucket:       cfg.S3.Bucket,
		S3AccessKey:    cfg.S3.AccessKey,
		S3SecretKey:    cfg.S3.SecretKey,
		S3UsePathStyle: cfg.S3.UsePathStyle,
		SignedURLBase:  api.MediaPath,
	}
	if cfg.Driver != media.DriverLocal {
		return options, nil
	}

	localPath, err := config.ResolveDataPath(cfg.Path, "")
	if err != nil {
		return media.Options{}, err
	}
	options.LocalPath = localPath

	return options, nil
}

// systemOptions maps the system settings onto the system service options.
func systemOptions(cfg config.SystemConfig) service.SystemOptions {
	return service.SystemOptions{
		ServerRestartCommand: cfg.ServerRestartCommand,
		ServerStopCommand:    cfg.ServerStopCommand,
		KioskRestartCommand:  cfg.KioskRestartCommand,
		KioskStopCommand:     cfg.KioskStopCommand,
		AllRestartCommand:    cfg.AllRestartCommand,
		AllStopCommand:       cfg.AllStopCommand,
	}
}
