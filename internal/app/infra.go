package app

import (
	"fmt"
	"os"

	"github.com/alexedwards/scs/v2"
	"github.com/rs/zerolog"

	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/cmdline"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/health"
	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/ratelimiter"
	"github.com/barthjs/photopi/internal/service"
	"github.com/barthjs/photopi/internal/session"
	"github.com/barthjs/photopi/internal/share"
	"github.com/barthjs/photopi/internal/update"
)

const sessionCookieName = config.AppName + "_session"

// infra holds the infrastructure of the application, taken from Deps or built from the configuration.
type infra struct {
	Bundle        *i18n.Bundle
	Mailer        *mailer.Mailer
	RateLimiter   ratelimiter.Limiter
	Session       *scs.SessionManager
	Storage       media.Storage
	Camera        camera.Camera
	CameraHealth  *health.Recorder
	Health        *health.Checker
	CloudProvider share.Provider
	Commands      service.CommandRunner
	Updater       service.Updater
}

// newInfra builds the infrastructure of the application from what Deps does not pass in
// and registers what it opens with the closers of the application.
func (a *App) newInfra(logger zerolog.Logger, cfg config.Config, deps Deps) (infra, error) {
	in := infra{
		Storage:       deps.Storage,
		Camera:        deps.Camera,
		CloudProvider: deps.CloudProvider,
		Commands:      deps.Commands,
		Updater:       deps.Updater,
	}

	var err error
	if in.Bundle, err = i18n.New(i18nOptions(logger, cfg)...); err != nil {
		return infra{}, fmt.Errorf("loading i18n catalogs: %w", err)
	}

	mailOptions := mailerOptions(cfg.Mail)
	mailOptions.Transport = deps.MailTransport
	if in.Mailer, err = mailer.New(logger, mailOptions, in.Bundle); err != nil {
		return infra{}, fmt.Errorf("initializing mailer: %w", err)
	}

	if in.RateLimiter, err = ratelimiter.New(); err != nil {
		return infra{}, fmt.Errorf("initializing rate limiter: %w", err)
	}

	if in.Session, err = session.New(sessionCookieName, cfg.Server.SSLEnabled); err != nil {
		return infra{}, fmt.Errorf("initializing session manager: %w", err)
	}
	sessions := in.Session
	a.closers = append(a.closers, closerFunc(func() error {
		session.StopCleanup(sessions)
		return nil
	}))

	if in.Storage == nil {
		if in.Storage, err = NewStorage(logger, cfg); err != nil {
			return infra{}, err
		}
	}

	if in.Camera == nil {
		if in.Camera, err = camera.New(logger, cameraOptions(cfg.Camera)); err != nil {
			return infra{}, fmt.Errorf("initializing camera: %w", err)
		}
	}

	in.CameraHealth = health.NewRecorder()
	// Without the camera, the gallery and the shared links keep working,
	// so a failing camera degrades the application instead of making it unavailable.
	in.Health = health.NewChecker(deps.Version, nil, health.WithOptional("camera", in.CameraHealth))

	if in.CloudProvider == nil {
		if in.CloudProvider, err = share.New(cloudProviderOptions(cfg.Share.Cloud)); err != nil {
			return infra{}, fmt.Errorf("initializing cloud share provider: %w", err)
		}
	}

	if in.Commands == nil {
		in.Commands = service.NewExecRunner(logger)
	}

	if in.Updater == nil {
		in.Updater = NewUpdater(deps.Version)
	}

	return in, nil
}

// NewStorage builds the configured file storage.
func NewStorage(logger zerolog.Logger, cfg config.Config) (media.Storage, error) {
	options, err := StorageOptions(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("resolving the storage path: %w", err)
	}

	storage, err := media.New(options)
	if err != nil {
		return nil, fmt.Errorf("initializing media storage: %w", err)
	}
	if options.Driver == media.DriverLocal {
		logger.Info().Str("path", options.LocalPath).Msg("using local storage")
	}
	return storage, nil
}

// NewUpdater returns the updater for the releases of the application,
// picking the archives of the architecture the binary was built for.
func NewUpdater(version string) *update.Updater {
	return update.New(update.Options{
		Repository: config.AppRepository,
		Binary:     config.AppName,
		Version:    version,
		Arch:       cmdline.ResolveArch(),
		Token:      os.Getenv("GITHUB_TOKEN"),
	})
}
