// Package app assembles the application from its configuration. New builds the whole application,
// and the entry point and the feature tests both use it, so they run the same wiring.
// Deps replaces the parts a test cannot use. Commands that need only some parts use the building blocks New is made of.
package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
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
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/health"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/ratelimiter"
	"github.com/barthjs/photopi/internal/service"
	"github.com/barthjs/photopi/internal/share"
	"github.com/barthjs/photopi/internal/ws"
)

// rateLimitCleanupInterval is how often expired rate limit entries are removed.
const rateLimitCleanupInterval = 15 * time.Minute

// Deps holds what New cannot derive from the configuration, where Stop is required
// and a zero field selects the production default.
type Deps struct {
	// ConfigFilePath is where a saved configuration is written, empty if no config file was found.
	ConfigFilePath string
	// ConfigSources reports by JSON path where each setting was loaded from, see structconf.PathSources.
	ConfigSources map[string]structconf.Source
	// OnUpdated runs once a newer release is installed, to restart into it.
	OnUpdated func()
	// Stop ends the application.
	Stop context.CancelFunc
	// Version is reported by the health endpoints and the public metadata.
	Version string

	// Camera replaces the camera selected by the configuration.
	Camera camera.Camera
	// CloudProvider replaces the cloud share provider selected by the configuration.
	CloudProvider share.Provider
	// Commands replaces the runner of external programs.
	Commands service.CommandRunner
	// MailTransport replaces the transport selected by the mail driver.
	MailTransport mailer.Transport
	// Storage replaces the storage selected by the configuration.
	Storage media.Storage
	// Updater installs newer releases. Nil builds one for Version, see NewUpdater.
	Updater service.Updater
}

// App is the assembled application.
type App struct {
	handler http.Handler

	rateLimiter    ratelimiter.Limiter
	hub            *ws.Hub
	captureService *service.CaptureService
	shareService   *service.ShareService

	// background tracks the work Start runs and the work requests start, so Close can wait for it to finish.
	background sync.WaitGroup
	// closers release what New opened, in the order New opened it.
	closers   []io.Closer
	closeOnce sync.Once
}

// New assembles the application without starting background work, see Start, and closes what it opened when it fails.
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

	in, err := a.newInfra(logger, cfg, deps)
	if err != nil {
		return nil, err
	}
	a.rateLimiter = in.RateLimiter
	var server *api.API
	a.hub = ws.New(logger, func() ([][]byte, error) { return realtimeSnapshot(deps.Version, server, in.Health) })
	stores := NewStores(in.Storage)

	shareSigningKey, err := selfShareSigningKey(cfg.App.Secret)
	if err != nil {
		return nil, err
	}

	captureService := service.NewCaptureService(logger, captureOptions(cfg.Capture, cfg.Camera), stores.Run, stores.Capture, stores.Overlay)
	a.captureService = captureService

	layoutService := service.NewLayoutService(stores.Layout)
	if err := layoutService.EnsureDefaults(ctx, in.Bundle, string(cfg.App.Language)); err != nil {
		return nil, fmt.Errorf("seeding default print layouts: %w", err)
	}

	printService := service.NewPrintService(
		logger,
		printOptions(cfg.Print, cfg.Capture),
		in.Commands,
		stores.Run,
		stores.Capture,
		stores.Collage,
		stores.Layout,
	)
	shareService := service.NewShareService(
		logger,
		shareOptions(cfg, shareSigningKey),
		in.Mailer,
		in.CloudProvider,
		stores.Run,
		captureService,
		printService,
	)
	a.shareService = shareService

	server = api.New(ctx, api.Deps{
		Options: apiOptions(cfg),
		Logger:  logger,
		WG:      &a.background,

		Health:      in.Health,
		Hub:         a.hub,
		RateLimiter: in.RateLimiter,
		Session:     in.Session,

		Camera:       in.Camera,
		CameraHealth: in.CameraHealth,
		Mailer:       in.Mailer,
		MediaHandler: media.SignedHandler(in.Storage),

		AuthService:    service.NewAuthService(authOptions(cfg.App), in.Session),
		CaptureService: captureService,
		ConfigService: service.NewConfigService(cfg, service.ConfigOptions{
			FilePath: deps.ConfigFilePath,
			Sources:  deps.ConfigSources,
		}),
		GalleryService:    service.NewGalleryService(stores.Run, stores.Capture, captureService),
		LayoutService:     layoutService,
		OverlayService:    service.NewOverlayService(overlayOptions(cfg.Camera), stores.Overlay),
		PrintService:      printService,
		RunSessionService: service.NewRunSessionService(in.Session),
		ShareService:      shareService,
		SystemService:     service.NewSystemService(systemOptions(cfg.System), in.Health, in.Commands, deps.Stop),
		UpdateService:     service.NewUpdateService(logger, in.Updater, deps.OnUpdated),
	})
	a.handler = server.Routes()
	// The snapshot of a new client sends the states the checker found last, so it finds them once now
	in.Health.Refresh()

	return a, nil
}

// Handler returns the HTTP handler of the application.
func (a *App) Handler() http.Handler {
	return a.handler
}

// Start runs the background work until ctx is canceled.
func (a *App) Start(ctx context.Context) {
	a.background.Go(func() { a.hub.Run(ctx) })
	a.background.Go(func() { a.shareService.RunCloudCheck(ctx) })
	a.background.Go(func() { a.captureService.PrepareOverlay(ctx) })
	a.background.Go(func() { ratelimiter.RunCleanup(ctx, a.rateLimiter, rateLimitCleanupInterval) })
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

// selfShareSigningKey returns the key that signs the self share links, derived from the app secret,
// so the links survive a restart, or a random one for the lifetime of the process.
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

// realtimeSnapshot returns the messages a newly connected client receives:
// the greeting with the version and the settings of the application, and the state of its dependencies.
func realtimeSnapshot(version string, server *api.API, checker *health.Checker) ([][]byte, error) {
	hello, err := json.Marshal(domain.NewMessage(domain.MessageHello, domain.Hello{
		Version: version,
		Meta:    server.MetaRevision(),
	}))
	if err != nil {
		return nil, err
	}

	checks, err := json.Marshal(domain.NewMessage(domain.MessageHealth, checker.Reported()))
	if err != nil {
		return nil, err
	}

	return [][]byte{
		hello,
		checks,
	}, nil
}

// broadcast sends a message to every connected client. A failed message is only logged,
// since the clients receive the current state again on their next connect.
func broadcast(logger zerolog.Logger, hub *ws.Hub, message domain.Message) {
	data, err := json.Marshal(message)
	if err != nil {
		logger.Error().Err(err).Str("type", string(message.Type)).Msg("failed to encode message")
		return
	}
	hub.Broadcast(data)
}
