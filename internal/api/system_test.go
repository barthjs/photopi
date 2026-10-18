package api_test

import (
	"io/fs"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/web"
)

func TestSystemAccess(t *testing.T) {
	t.Parallel()

	routes := []accessRoute{
		{
			method: http.MethodGet,
			path:   "/healthz",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/readyz",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/meta",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/ping",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/system/info",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/server/restart",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/server/stop",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/kiosk/restart",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/kiosk/stop",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/all/restart",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/all/stop",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/process/exit",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/host/reboot",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/system/host/shutdown",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
	}

	testAccess(t, routes)

	t.Run("runs nothing for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		for _, route := range routes {
			h.Client.Do(route.method, route.path, route.body).AssertStatus(route.statuses[clientGuest])
		}

		assert.Zero(t, h.Commands.count())
		assert.False(t, h.stopped())
	})
}

func TestHealth_Live(t *testing.T) {
	t.Parallel()

	t.Run("reports the liveness", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.GET("/healthz").AssertOK().AssertJSONPath("status", "ok").AssertJSONPath("version", harnessVersion)
	})
}

func TestHealth_Ready(t *testing.T) {
	t.Parallel()

	t.Run("reports the dependencies", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.GET("/readyz").AssertOK().AssertJSONPath("status", "ok").AssertJSONPath("checks.camera", "ok")
	})
}

func TestAPI_Routes(t *testing.T) {
	t.Parallel()

	t.Run("serves the docs in production", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.App.Env = config.EnvProduction
		}))

		h.Client.GET("/api/docs").AssertStatus(http.StatusMovedPermanently).AssertHeader("Location", "/api/docs/")
		h.Client.GET("/api/docs/").AssertOK()

		spec := h.Client.GET("/api/docs/doc.json").AssertOK()
		assert.Contains(t, string(spec.Body), `"swagger"`)
		assert.NotContains(t, string(spec.Body), "internal_", "the schemas carry their type names without the package path")
	})

	t.Run("returns not found for an unknown route", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.GET("/api/"+h.Faker.LetterN(12)).AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("rejects a wrong method", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.DELETE("/api/ping").AssertStatus(http.StatusMethodNotAllowed).AssertJSONPath("code", "method_not_allowed")
	})
}

func TestMetaGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the public settings", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, func(h *harness) {
			h.Config.App.Name = h.Faker.AppName()
			h.Config.Share.Email.Enabled = false
		})

		response := h.Client.GET("/api/meta").AssertOK()
		meta := response.Decode[api.MetaResponse]()

		assert.Equal(t, h.Config.App.Name, meta.Name)
		assert.Equal(t, harnessVersion, meta.Version)
		assert.Equal(t, h.Config.Capture.ImagesPerRun, meta.Capture.ImagesPerRun)
		assert.False(t, meta.Share.EmailEnabled)
		assert.False(t, meta.Share.CloudEnabled)
		assert.False(t, meta.Share.PrintEnabled)
		assert.NotContains(t, string(response.Body), h.Config.App.Secret, "the public settings never carry the secret")
	})
}

func TestPingGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("answers pong", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/ping").AssertOK().AssertHeader("Content-Type", "text/plain")
		assert.Equal(t, "pong", string(response.Body))
	})
}

func TestSystemInfoGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("reports the host and the commands", func(t *testing.T) {
		t.Parallel()
		command := ""
		h := newHarness(t, withoutSystemCommands(), func(h *harness) {
			command = h.Faker.LoremIpsumSentence(6)
			h.Config.System.KioskRestartCommand = command
		})

		info := h.asAdmin().GET("/api/system/info").AssertOK().Decode[api.SystemInfoResponse]()

		assert.Equal(t, harnessVersion, info.Version)
		assert.Positive(t, info.MemTotal)
		assert.Positive(t, info.DiskTotal)
		assert.Equal(t, api.SystemCommandsResponse{
			KioskRestart: command,
		}, info.Commands, "an empty command is an action that is not configured")
	})
}

func TestSystemServerRestartHandler(t *testing.T) {
	t.Parallel()

	testSystemActionHandler(t, "/api/system/server/restart", func(c *config.SystemConfig, command string) {
		c.ServerRestartCommand = command
	})
	testRejectedDuringUpdate(t, "/api/system/server/restart")
}

func TestSystemServerStopHandler(t *testing.T) {
	t.Parallel()

	testSystemActionHandler(t, "/api/system/server/stop", func(c *config.SystemConfig, command string) {
		c.ServerStopCommand = command
	})
	testRejectedDuringUpdate(t, "/api/system/server/stop")
}

func TestSystemKioskRestartHandler(t *testing.T) {
	t.Parallel()

	testSystemActionHandler(t, "/api/system/kiosk/restart", func(c *config.SystemConfig, command string) {
		c.KioskRestartCommand = command
	})
}

func TestSystemKioskStopHandler(t *testing.T) {
	t.Parallel()

	testSystemActionHandler(t, "/api/system/kiosk/stop", func(c *config.SystemConfig, command string) {
		c.KioskStopCommand = command
	})
}

func TestSystemAllRestartHandler(t *testing.T) {
	t.Parallel()

	testSystemActionHandler(t, "/api/system/all/restart", func(c *config.SystemConfig, command string) {
		c.AllRestartCommand = command
	})
	testRejectedDuringUpdate(t, "/api/system/all/restart")
}

func TestSystemAllStopHandler(t *testing.T) {
	t.Parallel()

	testSystemActionHandler(t, "/api/system/all/stop", func(c *config.SystemConfig, command string) {
		c.AllStopCommand = command
	})
	testRejectedDuringUpdate(t, "/api/system/all/stop")
}

// testSystemActionHandler runs the subtests of an endpoint that runs the command set up by configure.
func testSystemActionHandler(t *testing.T, path string, configure func(*config.SystemConfig, string)) {
	t.Helper()

	t.Run("runs the configured command", func(t *testing.T) {
		t.Parallel()
		var command string
		h := newHarness(t, withoutSystemCommands(), func(h *harness) {
			command = h.Faker.LoremIpsumSentence(6)
			configure(&h.Config.System, command)
		})

		h.asAdmin().POST(path, nil).AssertNoContent()

		calls := h.Commands.callsOf("sh")
		require.Len(t, calls, 1)
		assert.Equal(t, []string{"-c", command}, calls[0].Args)
	})

	t.Run("rejects an action without a command", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withoutSystemCommands())

		h.asAdmin().POST(path, nil).AssertBadRequest()

		assert.Zero(t, h.Commands.count())
	})

	t.Run("reports a failed start", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withoutSystemCommands(), func(h *harness) { configure(&h.Config.System, h.Faker.LoremIpsumSentence(6)) })
		h.Commands.answer("sh", "", errFake)

		h.asAdmin().POST(path, nil).AssertStatus(http.StatusInternalServerError)
	})
}

// withoutSystemCommands clears the commands the default configuration sets up for systemd.
func withoutSystemCommands() option {
	return withConfig(func(cfg *config.Config) { cfg.System = config.SystemConfig{} })
}

func TestSystemProcessExitHandler(t *testing.T) {
	t.Parallel()

	t.Run("stops the application", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.asAdmin().POST("/api/system/process/exit", nil).AssertNoContent()

		assert.True(t, h.stopped())
		assert.Zero(t, h.Commands.count())
	})
	testRejectedDuringUpdate(t, "/api/system/process/exit")
}

func TestSystemHostRebootHandler(t *testing.T) {
	t.Parallel()

	t.Run("reboots the host", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.asAdmin().POST("/api/system/host/reboot", nil).AssertNoContent()

		if runtime.GOOS == "windows" {
			calls := h.Commands.callsOf("shutdown")
			require.Len(t, calls, 1)
			assert.Equal(t, []string{"/r", "/t", "0"}, calls[0].Args)
			return
		}
		require.Len(t, h.Commands.callsOf("reboot"), 1)
	})
	testRejectedDuringUpdate(t, "/api/system/host/reboot")
}

func TestSystemHostShutdownHandler(t *testing.T) {
	t.Parallel()

	t.Run("shuts down the host", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.asAdmin().POST("/api/system/host/shutdown", nil).AssertNoContent()

		calls := h.Commands.callsOf("shutdown")
		require.Len(t, calls, 1)
		if runtime.GOOS == "windows" {
			assert.Equal(t, []string{"/s", "/t", "0"}, calls[0].Args)
			return
		}
		assert.Equal(t, []string{"-h", "now"}, calls[0].Args)
	})
	testRejectedDuringUpdate(t, "/api/system/host/shutdown")
}

// testRejectedDuringUpdate runs the subtest of an endpoint that ends the process and waits for a running update.
func testRejectedDuringUpdate(t *testing.T, path string) {
	t.Helper()

	t.Run("rejects the action while an update is being installed", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withUpdater(newInstallableUpdater(t)))
		admin := h.asAdmin()
		admin.POST("/api/update", nil).AssertStatus(http.StatusAccepted)

		admin.POST(path, nil).AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")

		assert.Zero(t, h.Commands.count())
		assert.False(t, h.stopped())
	})
}

func TestAPI_StaticRoutes(t *testing.T) {
	t.Parallel()

	t.Run("serves the pages", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		index := h.Client.GET("/").AssertOK()
		assert.Contains(t, string(index.Body), "<html")

		// Unknown paths outside the API belong to the client-side router and get the index page too
		h.Client.GET("/"+h.Faker.LetterN(10)).AssertOK().AssertHeader("Content-Type", index.Header.Get("Content-Type"))
	})

	t.Run("serves the public files", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		entries, err := fs.ReadDir(web.Files, "dist")
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() || strings.HasSuffix(entry.Name(), ".html") {
				continue
			}
			want, err := web.Files.ReadFile("dist/" + entry.Name())
			require.NoError(t, err)

			response := h.Client.GET("/" + entry.Name()).AssertOK()

			assert.Equal(t, string(want), string(response.Body), entry.Name())
		}
	})

	t.Run("caches the assets", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		assets, err := fs.ReadDir(web.Files, "dist/assets")
		require.NoError(t, err)
		require.NotEmpty(t, assets, "the web interface is built into the binary")

		h.Client.GET("/assets/"+assets[0].Name()).AssertOK().AssertHeader("Cache-Control", "public, max-age=31536000, immutable")
	})
}
