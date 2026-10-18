// Package app assembles the application from its configuration. New builds the whole application,
// and the entry point and the feature tests both use it, so they run the same wiring.
// Deps replaces the parts a test cannot use. Commands that need only some parts use the building blocks New is made of.
package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/config/structconf"
	"github.com/barthjs/photopi/internal/health"
	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/ratelimiter"
	"github.com/barthjs/photopi/internal/service"
	"github.com/barthjs/photopi/internal/session"
	"github.com/barthjs/photopi/internal/share"
)

// rateLimitCleanupInterval is how often expired rate limit entries are removed.
const rateLimitCleanupInterval = 15 * time.Minute

// Deps holds what New cannot derive from the configuration. Stop is required,
// every other field is optional, and a zero value selects the production default.
type Deps struct {
	// Stop ends the application, for example after a self-update or a request to exit the process.
	Stop context.CancelFunc
	// Version is reported by the health endpoints and the public metadata.
	Version string
	// ConfigFilePath is where a saved configuration is written, empty if no config file was found.
	ConfigFilePath string
	// ConfigSources reports by JSON path where each setting was loaded from, see structconf.PathSources.
	ConfigSources map[string]structconf.Source

	// Storage replaces the storage selected by the configuration.
	Storage media.Storage
	// Camera replaces the camera selected by the configuration.
	Camera camera.Camera
	// MailTransport replaces the transport selected by the mail driver.
	MailTransport mailer.Transport
	// CloudProvider replaces the cloud share provider selected by the configuration.
	CloudProvider share.Provider
	// Commands replaces the runner of external programs.
	Commands service.CommandRunner
	// Updater installs newer releases. Nil builds one for Version, see NewUpdater.
	Updater service.Updater
	// OnUpdated runs once a newer release is installed, to restart into it.
	OnUpdated func()
}

// App is the assembled application.
type App struct {
	handler http.Handler

	rateLimiter    ratelimiter.Limiter
	captureService *service.CaptureService
	shareService   *service.ShareService

	// background tracks the work Start runs and the work requests start, so Close can wait for it to finish.
	background sync.WaitGroup
	// closers release what New opened, in the order New opened it.
	closers   []io.Closer
	closeOnce sync.Once
}

// New assembles the application, but starts no background work, see Start.
// If it fails, it closes what it opened so far.
func New(ctx context.Context, logger zerolog.Logger, cfg config.Config, deps Deps) (_ *App, err error) {
	if deps.Stop == nil {
		return nil, errors.New("Deps.Stop is required")
	}
	a := &App{}
	defer func() {
		if err != nil {
			_ = a.Close()
		}
	}()

	if deps.Commands == nil {
		deps.Commands = service.NewExecRunner(logger)
	}

	i18nBundle, err := i18n.New(i18nOptions(logger, cfg)...)
	if err != nil {
		return nil, fmt.Errorf("loading i18n catalogs: %w", err)
	}

	mailOptions := mailerOptions(cfg.Mail)
	mailOptions.Transport = deps.MailTransport
	mailService, err := mailer.New(logger, mailOptions, i18nBundle)
	if err != nil {
		return nil, fmt.Errorf("initializing mailer: %w", err)
	}

	rl, err := ratelimiter.New()
	if err != nil {
		return nil, fmt.Errorf("initializing rate limiter: %w", err)
	}
	a.rateLimiter = rl

	storage := deps.Storage
	if storage == nil {
		storage, err = NewStorage(logger, cfg)
		if err != nil {
			return nil, err
		}
	}

	shareSigningKey, err := selfShareSigningKey(cfg.App.Secret)
	if err != nil {
		return nil, err
	}

	cam := deps.Camera
	if cam == nil {
		cam, err = camera.New(logger, cameraOptions(cfg.Camera))
		if err != nil {
			return nil, fmt.Errorf("initializing camera: %w", err)
		}
	}

	cloudProvider := deps.CloudProvider
	if cloudProvider == nil {
		cloudProvider, err = share.New(cloudProviderOptions(cfg.Share.Cloud))
		if err != nil {
			return nil, fmt.Errorf("initializing cloud share provider: %w", err)
		}
	}

	cameraHealth := health.NewRecorder()
	// Without the camera, the gallery and the shared links keep working,
	// so a failing camera degrades the application instead of making it unavailable.
	healthChecker := health.NewChecker(deps.Version, nil, health.WithOptional("camera", cameraHealth))

	sm, err := session.New(cfg.Server.SSLEnabled)
	if err != nil {
		return nil, fmt.Errorf("initializing session manager: %w", err)
	}
	a.closers = append(a.closers, closerFunc(func() error {
		session.StopCleanup(sm)
		return nil
	}))

	authService := service.NewAuthService(authOptions(cfg.App), sm)
	configService := service.NewConfigService(cfg, service.ConfigOptions{
		FilePath: deps.ConfigFilePath,
		Sources:  deps.ConfigSources,
	})
	stores := NewStores(storage)
	captureService := service.NewCaptureService(logger, captureOptions(cfg.Capture, cfg.Camera), stores.Run, stores.Capture, stores.Overlay)
	a.captureService = captureService
	galleryService := service.NewGalleryService(stores.Run, stores.Capture, captureService)
	overlayService := service.NewOverlayService(overlayOptions(cfg.Camera), stores.Overlay)
	systemService := service.NewSystemService(systemOptions(cfg.System), healthChecker, deps.Commands, deps.Stop)
	updater := deps.Updater
	if updater == nil {
		updater = NewUpdater(deps.Version)
	}
	updateService := service.NewUpdateService(logger, updater, deps.OnUpdated)

	layoutService := service.NewLayoutService(stores.Layout)
	if err := layoutService.EnsureDefaults(ctx, i18nBundle, string(cfg.App.Language)); err != nil {
		return nil, fmt.Errorf("seeding default print layouts: %w", err)
	}

	printService := service.NewPrintService(
		logger,
		printOptions(cfg.Print, cfg.Capture),
		deps.Commands,
		stores.Run,
		stores.Capture,
		stores.Collage,
		stores.Layout,
	)
	shareService := service.NewShareService(
		logger,
		shareOptions(cfg, shareSigningKey),
		mailService,
		cloudProvider,
		stores.Run,
		captureService,
		printService,
	)
	a.shareService = shareService

	a.handler = api.New(ctx, api.Deps{
		Options: apiOptions(cfg),
		Logger:  logger,
		WG:      &a.background,

		Health:      healthChecker,
		RateLimiter: rl,
		Session:     sm,

		Camera:       cam,
		CameraHealth: cameraHealth,
		Mailer:       mailService,
		MediaHandler: media.SignedHandler(storage),

		AuthService:       authService,
		CaptureService:    captureService,
		ConfigService:     configService,
		GalleryService:    galleryService,
		LayoutService:     layoutService,
		OverlayService:    overlayService,
		PrintService:      printService,
		RunSessionService: service.NewRunSessionService(sm),
		ShareService:      shareService,
		SystemService:     systemService,
		UpdateService:     updateService,
	}).Routes()

	return a, nil
}

// Handler returns the HTTP handler of the application.
func (a *App) Handler() http.Handler {
	return a.handler
}

// Start runs the background work until ctx is canceled.
func (a *App) Start(ctx context.Context) {
	a.background.Go(func() { ratelimiter.RunCleanup(ctx, a.rateLimiter, rateLimitCleanupInterval) })
	a.background.Go(func() { a.shareService.RunCloudCheck(ctx) })
	a.background.Go(func() { a.captureService.PrepareOverlay(ctx) })
}

// Close waits for the background work to end, the work of Start and the work requests started,
// and releases what New opened. Cancel the context passed to Start first, otherwise Close blocks.
func (a *App) Close() error {
	a.background.Wait()
	// The capture service runs the captures of requests on goroutines of its own
	if a.captureService != nil {
		a.captureService.Wait()
	}

	var errs []error
	a.closeOnce.Do(func() {
		for _, closer := range a.closers {
			errs = append(errs, closer.Close())
		}
	})
	return errors.Join(errs...)
}

// selfShareSigningKey returns the key that signs the self share links. It is derived from the app secret,
// so the links survive a restart. Without a secret, a random key is generated,
// and the links stay valid only for the lifetime of the process.
func selfShareSigningKey(secret string) ([]byte, error) {
	if secret != "" {
		return []byte(secret), nil
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating share signing key: %w", err)
	}
	return key, nil
}

// closerFunc adapts a function to io.Closer.
type closerFunc func() error

// Close calls the function.
func (f closerFunc) Close() error {
	return f()
}
