package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/rs/zerolog"

	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/health"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/ratelimiter"
	"github.com/barthjs/photopi/internal/service"
	"github.com/barthjs/photopi/internal/ws"
)

// MediaPath is where the signed URLs of the local storage are served, see media.SignedHandler.
const MediaPath = "/media"

// API holds the HTTP handlers and the services they call.
type API struct {
	// ctx ends with the application, so background work started by a request stops with it.
	ctx     context.Context
	hub     *ws.Hub
	options Options
	logger  zerolog.Logger
	now     func() time.Time
	wg      *sync.WaitGroup

	health      *health.Checker
	rateLimiter ratelimiter.Limiter
	sm          *scs.SessionManager

	camera       camera.Camera
	cameraHealth *health.Recorder
	mailer       *mailer.Mailer
	mediaHandler http.Handler

	authService       *service.AuthService
	captureService    *service.CaptureService
	configService     *service.ConfigService
	galleryService    *service.GalleryService
	layoutService     *service.LayoutService
	overlayService    *service.OverlayService
	printService      *service.PrintService
	runSessionService *service.RunSessionService
	shareService      *service.ShareService
	systemService     *service.SystemService
	updateService     *service.UpdateService
}

// Options holds the settings the API reads at runtime.
type Options struct {
	// Name is the application name reported to the frontend.
	Name string
	// Language is the language of the user interface.
	Language string
	// Theme holds the accent colors of the user interface.
	Theme ThemeSettings
	// SplashColor is the background of the splash screen of the installed app, the theme color when empty.
	SplashColor string
	// Camera holds the camera settings the frontend and the preview need.
	Camera CameraOptions
	// Capture holds the capture settings the frontend shows.
	Capture CaptureOptions
	// TrustedProxies may set the forwarding headers that name the client address.
	TrustedProxies []*net.IPNet
	// CORSAllowedOrigins may call the API from another domain, empty disables CORS.
	CORSAllowedOrigins []string
	// CORSAllowCredentials lets the allowed origins send cookies.
	CORSAllowCredentials bool
	// Docs serves the documentation with the API reference under /docs.
	Docs bool
}

// CameraOptions holds the camera settings the API reads.
type CameraOptions struct {
	Source camera.Source
	// OverlayPath is the storage path of the overlay composited onto every capture, empty for none.
	OverlayPath string
}

// CaptureOptions holds the capture settings the API reads.
type CaptureOptions struct {
	ImagesPerRun   int
	WelcomeMessage string
	// AdminEmail is shown to a guest when emailing their captures fails.
	AdminEmail string
}

// Deps are the options, infrastructure, and services the API is built from.
type Deps struct {
	Options Options
	Logger  zerolog.Logger
	// Now is the clock of the API, time.Now when nil.
	Now func() time.Time
	WG  *sync.WaitGroup

	Health      *health.Checker
	Hub         *ws.Hub
	RateLimiter ratelimiter.Limiter
	Session     *scs.SessionManager

	Camera       camera.Camera
	CameraHealth *health.Recorder
	Mailer       *mailer.Mailer
	// MediaHandler serves the signed URLs of the storage below MediaPath.
	MediaHandler http.Handler

	AuthService       *service.AuthService
	CaptureService    *service.CaptureService
	ConfigService     *service.ConfigService
	GalleryService    *service.GalleryService
	LayoutService     *service.LayoutService
	OverlayService    *service.OverlayService
	PrintService      *service.PrintService
	RunSessionService *service.RunSessionService
	ShareService      *service.ShareService
	SystemService     *service.SystemService
	UpdateService     *service.UpdateService
}

// New creates an API. ctx ends with the application.
func New(ctx context.Context, d Deps) *API {
	now := d.Now
	if now == nil {
		now = time.Now
	}

	return &API{
		ctx:     ctx,
		hub:     d.Hub,
		options: d.Options,
		logger:  d.Logger,
		now:     now,
		wg:      d.WG,

		health:       d.Health,
		cameraHealth: d.CameraHealth,
		rateLimiter:  d.RateLimiter,
		sm:           d.Session,

		camera:       d.Camera,
		mailer:       d.Mailer,
		mediaHandler: d.MediaHandler,

		authService:       d.AuthService,
		captureService:    d.CaptureService,
		configService:     d.ConfigService,
		galleryService:    d.GalleryService,
		layoutService:     d.LayoutService,
		overlayService:    d.OverlayService,
		printService:      d.PrintService,
		runSessionService: d.RunSessionService,
		shareService:      d.ShareService,
		systemService:     d.SystemService,
		updateService:     d.UpdateService,
	}
}

// background runs fn in a separate goroutine with panic recovery
// and a context that ends after timeout or with the application, whichever comes first.
// It adds a counter to the API.wg, so the application waits for it before exiting.
func (a *API) background(timeout time.Duration, fn func(ctx context.Context)) {
	a.wg.Go(func() {
		defer func() {
			if err := recover(); err != nil {
				a.logger.Error().Err(fmt.Errorf("%s", err)).Msg("panic in background task")
			}
		}()

		ctx, cancel := context.WithTimeout(a.ctx, timeout)
		defer cancel()

		fn(ctx)
	})
}
