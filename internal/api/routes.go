package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	gorillaHandlers "github.com/gorilla/handlers"

	"github.com/barthjs/photopi/docs"
	"github.com/barthjs/photopi/internal/server"
	"github.com/barthjs/photopi/web"
)

// contentSecurityPolicy only allows the scripts and connections of the application itself. Styles may be inline,
// since Nuxt UI injects style elements at runtime. Fonts may be data: URLs for the fonts inlined in the stylesheets.
// Images may also come from data: and blob: URLs for the previews of local files, and from any http or https origin,
// since the guest share page shows the signed URLs of the storage, which point to the S3 endpoint with the s3 driver.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob: http: https:; font-src 'self' data:; connect-src 'self'; object-src 'none'; " +
	"base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// docsContentSecurityPolicy is the policy of the documentation, which only loads its own resources
// and compiles WebAssembly for its search in the browser.
const docsContentSecurityPolicy = "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; " +
	"style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// permissionsPolicy allows the camera for the webcam capture and full screen for the kiosk,
// and denies the browser features the application does not use.
const permissionsPolicy = "accelerometer=(), camera=(self), display-capture=(), fullscreen=(self), geolocation=(), " +
	"gyroscope=(), hid=(), magnetometer=(), microphone=(), midi=(), payment=(), serial=(), usb=()"

// Routes returns the HTTP router for the API.
func (a *API) Routes() http.Handler {
	r := chi.NewRouter()

	// The request ID comes first, so every later log line and error response carries it.
	// The body limit is the default for every route, uploads raise it for themselves.
	r.Use(
		server.RequestID,
		a.recoverPanic,
		server.RealIP(a.options.TrustedProxies),
		server.SecurityHeaders(server.SecurityHeadersOptions{
			ContentSecurityPolicy: contentSecurityPolicy,
			PermissionsPolicy:     permissionsPolicy,
		}),
		server.CORS(server.CORSOptions{
			AllowedOrigins:   a.options.CORSAllowedOrigins,
			AllowCredentials: a.options.CORSAllowCredentials,
		}),
		server.BodyLimit(server.DefaultMaxBodySize),
	)

	r.Get("/healthz", a.health.Live)
	r.Get("/readyz", a.health.Ready)

	r.With(server.Timeout(downloadTimeout)).Handle(MediaPath+"/*", http.StripPrefix(MediaPath, a.mediaHandler))

	a.realtimeRoutes(r)
	a.apiRoutes(r)
	a.docsRoutes(r)
	a.staticRoutes(r)

	return r
}

// realtimeRoutes registers the public WebSocket endpoint outside the "/api" group,
// whose timeout and compression do not fit a connection that stays open.
func (a *API) realtimeRoutes(r chi.Router) {
	r.With(a.logRequest).Get("/api/ws", a.realtimeConnectHandler)
}

// apiRoutes registers the JSON API under "/api".
func (a *API) apiRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		// Every route of the JSON API is logged. Compression is set per group,
		// since the camera preview and the ZIP downloads stream and must not be compressed.
		// State-changing requests from another origin are rejected unless CORS allows that origin explicitly.
		r.Use(
			server.Timeout(server.DefaultRequestTimeout),
			a.logRequest,
			server.CrossOrigin(server.CrossOriginOptions{
				TrustedOrigins: a.options.CORSAllowedOrigins,
				Deny:           http.HandlerFunc(a.forbiddenResponse),
			}),
		)

		r.Group(func(r chi.Router) {
			r.Use(gorillaHandlers.CompressHandler)

			r.NotFound(a.notFoundResponse)
			r.MethodNotAllowed(a.methodNotAllowedResponse)

			r.Get("/meta", a.metaGetHandler)
			r.Get("/ping", a.pingGetHandler)
		})

		a.authRoutes(r)
		a.cameraRoutes(r)
		a.configRoutes(r)
		a.galleryRoutes(r)
		a.overlayRoutes(r)
		a.printRoutes(r)
		a.shareRoutes(r)
		a.systemRoutes(r)
		a.updateRoutes(r)
	})
}

// authRoutes registers the auth routes under "/auth".
func (a *API) authRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Use(gorillaHandlers.CompressHandler, a.sm.LoadAndSave)

		r.Get("/me", a.authStatusGetHandler)
		r.With(a.requireGuest).Post("/login", a.authLoginHandler)
		r.Post("/logout", a.authLogoutHandler)
	})
}

// cameraRoutes registers the camera routes under "/camera".
func (a *API) cameraRoutes(r chi.Router) {
	r.Route("/camera", func(r chi.Router) {
		r.Use(a.sm.LoadAndSave)

		r.Get("/overlay", a.cameraOverlayGetHandler)
		// The preview streams until the client disconnects, so it has no deadline
		r.With(a.requireActiveRun, server.Timeout(0)).Get("/preview", a.cameraPreviewGetHandler)

		r.Route("/runs", func(r chi.Router) {
			r.Use(gorillaHandlers.CompressHandler)

			r.Post("/", a.runCreateHandler)

			r.Route("/{run}", func(r chi.Router) {
				r.Use(a.requireRunSession)

				r.Delete("/", a.runAbortHandler)

				r.Get("/captures/{index}", a.captureGetHandler)
				r.With(server.BodyLimit(uploadBodyLimit(maxCaptureUploadMB)), server.Timeout(uploadTimeout)).Post("/captures", a.captureCreateHandler)
				r.Get("/captures/{index}/preview", a.capturePreviewGetHandler)
			})
		})
	})
}

// configRoutes registers the admin config routes under "/config".
func (a *API) configRoutes(r chi.Router) {
	r.Route("/config", func(r chi.Router) {
		r.Use(gorillaHandlers.CompressHandler, a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/", a.configGetHandler)
		r.Put("/", a.configSaveHandler)

		r.Post("/storage/test", a.configStorageTestHandler)
		r.Post("/share/test", a.configShareTestHandler)
	})
}

// galleryRoutes registers the admin gallery routes under "/gallery".
func (a *API) galleryRoutes(r chi.Router) {
	r.Route("/gallery", func(r chi.Router) {
		r.Use(a.sm.LoadAndSave, a.requireAuthentication)

		r.With(gorillaHandlers.CompressHandler).Get("/", a.galleryListHandler)
		r.With(gorillaHandlers.CompressHandler).Delete("/{run}", a.galleryDeleteHandler)
		r.With(gorillaHandlers.CompressHandler).Post("/bulk", a.galleryBulkHandler)
		r.With(server.Timeout(eventDownloadTimeout)).Get("/zip", a.galleryZipHandler)
		r.With(server.Timeout(downloadTimeout)).Get("/{run}/zip", a.galleryRunZipHandler)

		r.With(gorillaHandlers.CompressHandler).Get("/days", a.galleryDayListHandler)

		r.With(gorillaHandlers.CompressHandler).Get("/events", a.galleryEventListHandler)
		r.With(server.Timeout(eventDownloadTimeout)).Get("/events/{event}/zip", a.galleryEventZipHandler)

		// The images are JPEGs, which compression does not shrink any further
		r.Get("/{run}/captures/{index}", a.galleryCaptureGetHandler)
		r.Get("/{run}/originals/{index}", a.galleryOriginalGetHandler)
		r.Get("/{run}/thumbnail", a.galleryThumbnailGetHandler)
	})
}

// overlayRoutes registers the overlay routes under "/overlays".
func (a *API) overlayRoutes(r chi.Router) {
	r.Route("/overlays", func(r chi.Router) {
		r.Use(gorillaHandlers.CompressHandler, a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/", a.overlayListHandler)
		r.Get("/{name}", a.overlayGetHandler)
		r.With(server.BodyLimit(uploadBodyLimit(maxOverlayUploadMB)), server.Timeout(uploadTimeout)).Post("/", a.overlayCreateHandler)
		r.Delete("/{name}", a.overlayDeleteHandler)
	})
}

// printRoutes registers the admin print routes under "/print".
func (a *API) printRoutes(r chi.Router) {
	r.Route("/print", func(r chi.Router) {
		r.Use(gorillaHandlers.CompressHandler, a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/printers", a.printerListHandler)
		r.Get("/layouts", a.layoutListHandler)
		r.Get("/layouts/{name}", a.layoutGetHandler)
		r.Put("/layouts/{name}", a.layoutSaveHandler)
		r.Delete("/layouts/{name}", a.layoutDeleteHandler)
		r.Post("/test", a.printTestHandler)
		r.Post("/resume", a.printResumeHandler)
		r.Post("/cancel", a.printCancelHandler)
	})
}

// shareRoutes registers the share routes under "/share".
func (a *API) shareRoutes(r chi.Router) {
	r.Route("/share", func(r chi.Router) {
		r.Use(gorillaHandlers.CompressHandler, a.sm.LoadAndSave)

		r.Route("/{run}", func(r chi.Router) {
			r.Use(a.requireRunSession)

			r.Get("/cloud-upload", a.shareCloudUploadGetHandler)
			// Per-run, not per-IP: this is the guest's retry budget for that run
			r.With(a.rateLimitPerRun("share:cloud-upload", shareRateLimit, shareRateLimitWindow)).Post("/cloud-upload", a.shareCloudUploadHandler)

			r.Get("/email", a.shareEmailGetHandler)
			r.With(a.rateLimitPerRun("share:email", shareRateLimit, shareRateLimitWindow)).Post("/email", a.shareEmailHandler)

			r.Get("/print", a.sharePrintGetHandler)
			r.With(a.rateLimitPerRun("share:print", shareRateLimit, shareRateLimitWindow)).Post("/print", a.sharePrintHandler)
		})

		r.Route("/public/{token}", func(r chi.Router) {
			r.Get("/", a.publicShareGetHandler)
			r.With(server.Timeout(downloadTimeout)).Get("/zip", a.publicShareZipHandler)
		})
	})
}

// systemRoutes registers the admin system routes under "/system".
func (a *API) systemRoutes(r chi.Router) {
	r.Route("/system", func(r chi.Router) {
		r.Use(gorillaHandlers.CompressHandler, a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/info", a.systemInfoGetHandler)

		r.Route("/server", func(r chi.Router) {
			r.Use(a.rejectDuringUpdate)

			r.Post("/restart", a.systemServerRestartHandler)
			r.Post("/stop", a.systemServerStopHandler)
		})

		r.Route("/kiosk", func(r chi.Router) {
			r.Post("/restart", a.systemKioskRestartHandler)
			r.Post("/stop", a.systemKioskStopHandler)
		})

		r.Route("/all", func(r chi.Router) {
			r.Use(a.rejectDuringUpdate)

			r.Post("/restart", a.systemAllRestartHandler)
			r.Post("/stop", a.systemAllStopHandler)
		})

		r.With(a.rejectDuringUpdate).Post("/process/exit", a.systemProcessExitHandler)

		r.Route("/host", func(r chi.Router) {
			r.Use(a.rejectDuringUpdate)

			r.Post("/reboot", a.systemHostRebootHandler)
			r.Post("/shutdown", a.systemHostShutdownHandler)
		})
	})
}

// updateRoutes registers the admin routes of the self-update under "/update".
func (a *API) updateRoutes(r chi.Router) {
	r.Route("/update", func(r chi.Router) {
		r.Use(gorillaHandlers.CompressHandler, a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/status", a.updateStatusGetHandler)
		r.Get("/", a.updateCheckHandler)
		r.Post("/", a.updateStartHandler)
	})
}

// docsRoutes serves the embedded documentation with the API reference under "/docs".
// Its bundles carry a hash in their names, so they are cached like the assets of the SPA.
func (a *API) docsRoutes(r chi.Router) {
	if !a.options.Docs {
		return
	}

	handler := http.StripPrefix("/docs", docs.Handler("/docs/"))
	secured := r.With(
		server.SecurityHeaders(server.SecurityHeadersOptions{
			ContentSecurityPolicy: docsContentSecurityPolicy,
			PermissionsPolicy:     permissionsPolicy,
		}),
		gorillaHandlers.CompressHandler,
	)

	r.Get("/docs", http.RedirectHandler("/docs/", http.StatusMovedPermanently).ServeHTTP)
	secured.With(a.cacheHeaderMiddleware).Handle("/docs/_astro/*", handler)
	secured.Handle("/docs/*", handler)
}

// staticRoutes serves the embedded SPA.
func (a *API) staticRoutes(r chi.Router) {
	staticFS, _ := fs.Sub(web.Files, "dist")
	fileServer := http.FileServerFS(staticFS)

	r.With(gorillaHandlers.CompressHandler, a.cacheHeaderMiddleware).Handle("/assets/*", fileServer)
	r.With(gorillaHandlers.CompressHandler, a.noCacheHeaderMiddleware).Handle("/sw.js", fileServer)
	r.With(gorillaHandlers.CompressHandler).Get("/site.webmanifest", a.manifestGetHandler)
	r.With(gorillaHandlers.CompressHandler).Get("/", a.indexGetHandler)

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		cleanPath := strings.TrimPrefix(req.URL.Path, "/")
		if stat, err := fs.Stat(staticFS, cleanPath); err == nil && !stat.IsDir() {
			gorillaHandlers.CompressHandler(fileServer).ServeHTTP(w, req)
			return
		}
		gorillaHandlers.CompressHandler(http.HandlerFunc(a.indexGetHandler)).ServeHTTP(w, req)
	})
}

// indexGetHandler serves the index.html of the SPA.
func (a *API) indexGetHandler(w http.ResponseWriter, _ *http.Request) {
	rawFile, _ := web.Files.ReadFile("dist/index.html")
	_, _ = w.Write(rawFile)
}
