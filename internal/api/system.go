package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"net/http"
	"time"

	"github.com/barthjs/photopi/internal/service"
)

// metaGetHandler returns the public configuration of the application.
//
//	@Summary		Get metadata
//	@Description	Returns the public configuration of the application.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	MetaResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/meta [get]
func (a *API) metaGetHandler(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, r, http.StatusOK, a.meta(), nil)
}

type MetaResponse struct {
	Name     string        `json:"name"`
	Version  string        `json:"version"`
	Language string        `json:"language"`
	Camera   CameraMeta    `json:"camera"`
	Capture  CaptureMeta   `json:"capture"`
	Share    ShareMeta     `json:"share"`
	Theme    ThemeSettings `json:"theme"`
}

// MetaRevision returns a hash of the meta response, which changes with every setting the web interface reads.
func (a *API) MetaRevision() string {
	data, _ := json.Marshal(a.meta())
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

// meta returns the public settings of the application.
func (a *API) meta() MetaResponse {
	return MetaResponse{
		Name:     a.options.Name,
		Version:  a.health.Version(),
		Language: a.options.Language,
		Camera: CameraMeta{
			Source:     string(a.options.Camera.Source),
			HasOverlay: a.options.Camera.OverlayPath != "",
		},
		Capture: CaptureMeta{
			ImagesPerRun:   a.options.Capture.ImagesPerRun,
			WelcomeMessage: a.options.Capture.WelcomeMessage,
		},
		Share: ShareMeta{
			EmailEnabled: a.shareService.EmailEnabled(),
			CloudEnabled: a.shareService.CloudEnabled(),
			PrintEnabled: a.printService.Enabled(),
			AdminEmail:   a.options.Capture.AdminEmail,
		},
		Theme: a.options.Theme,
	}
}

type CameraMeta struct {
	Source     string `json:"source"`
	HasOverlay bool   `json:"hasOverlay"`
}

type CaptureMeta struct {
	ImagesPerRun   int    `json:"imagesPerRun"`
	WelcomeMessage string `json:"welcomeMessage"`
}

type ShareMeta struct {
	EmailEnabled bool   `json:"emailEnabled"`
	CloudEnabled bool   `json:"cloudEnabled"`
	PrintEnabled bool   `json:"printEnabled"`
	AdminEmail   string `json:"adminEmail"`
}

// pingGetHandler responds with "pong".
//
//	@Summary		Get ping
//	@Description	Returns "pong" if the API is up and running.
//	@Tags			system
//	@Produce		plain
//	@Success		200	{string}	string	"pong"
//	@Router			/ping [get]
func (a *API) pingGetHandler(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusOK, "pong", nil)
}

// systemInfoGetHandler reports host and process metrics for the admin dashboard.
//
//	@Summary		Get system info
//	@Description	Reports host and process metrics for the admin dashboard (Admin only).
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	SystemInfoResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/info [get]
func (a *API) systemInfoGetHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := a.systemService.Stats(r.Context())
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, SystemInfoResponse{
		Version:    stats.Version,
		UptimeSecs: int64(stats.Uptime.Round(time.Second).Seconds()),
		Hostname:   stats.Hostname,
		IPs:        stats.IPs,
		CPUPercent: stats.CPUPercent,
		MemUsed:    stats.MemUsed,
		MemTotal:   stats.MemTotal,
		DiskUsed:   stats.DiskUsed,
		DiskTotal:  stats.DiskTotal,
		Commands:   toSystemCommandsResponse(a.systemService.Options()),
	}, nil)
}

type SystemInfoResponse struct {
	Version    string   `json:"version"`
	UptimeSecs int64    `json:"uptimeSeconds"`
	Hostname   string   `json:"hostname"`
	IPs        []string `json:"ips"`
	CPUPercent float64  `json:"cpuPercent"`
	MemUsed    uint64   `json:"memUsed"`
	MemTotal   uint64   `json:"memTotal"`
	DiskUsed   uint64   `json:"diskUsed"`
	DiskTotal  uint64   `json:"diskTotal"`
	// Commands are the configured commands of the system actions, an empty one disables its action.
	Commands SystemCommandsResponse `json:"commands"`
}

type SystemCommandsResponse struct {
	ServerRestart string `json:"serverRestart"`
	ServerStop    string `json:"serverStop"`
	KioskRestart  string `json:"kioskRestart"`
	KioskStop     string `json:"kioskStop"`
	AllRestart    string `json:"allRestart"`
	AllStop       string `json:"allStop"`
}

// toSystemCommandsResponse maps the commands of the system actions to their response.
func toSystemCommandsResponse(o service.SystemOptions) SystemCommandsResponse {
	return SystemCommandsResponse{
		ServerRestart: o.ServerRestartCommand,
		ServerStop:    o.ServerStopCommand,
		KioskRestart:  o.KioskRestartCommand,
		KioskStop:     o.KioskStopCommand,
		AllRestart:    o.AllRestartCommand,
		AllStop:       o.AllStopCommand,
	}
}

// systemServerRestartHandler runs the configured system.server_restart_command.
//
//	@Summary		Restart server
//	@Description	Runs the configured system.server_restart_command. Disabled (400) when none is configured (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/server/restart [post]
func (a *API) systemServerRestartHandler(w http.ResponseWriter, r *http.Request) {
	a.runSystemCommand(w, r, a.systemService.ServerRestartEnabled(), "server restart is disabled: system.server_restart_command is not configured", a.systemService.RestartServer)
}

// systemServerStopHandler runs the configured system.server_stop_command.
//
//	@Summary		Stop server
//	@Description	Runs the configured system.server_stop_command. Disabled (400) when none is configured (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/server/stop [post]
func (a *API) systemServerStopHandler(w http.ResponseWriter, r *http.Request) {
	a.runSystemCommand(w, r, a.systemService.ServerStopEnabled(), "server stop is disabled: system.server_stop_command is not configured", a.systemService.StopServer)
}

// systemKioskRestartHandler runs the configured system.kiosk_restart_command.
//
//	@Summary		Restart kiosk
//	@Description	Runs the configured system.kiosk_restart_command. Disabled (400) when none is configured (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/kiosk/restart [post]
func (a *API) systemKioskRestartHandler(w http.ResponseWriter, r *http.Request) {
	a.runSystemCommand(w, r, a.systemService.KioskRestartEnabled(), "kiosk restart is disabled: system.kiosk_restart_command is not configured", a.systemService.RestartKiosk)
}

// systemKioskStopHandler runs the configured system.kiosk_stop_command.
//
//	@Summary		Stop kiosk
//	@Description	Runs the configured system.kiosk_stop_command. Disabled (400) when none is configured (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/kiosk/stop [post]
func (a *API) systemKioskStopHandler(w http.ResponseWriter, r *http.Request) {
	a.runSystemCommand(w, r, a.systemService.KioskStopEnabled(), "kiosk stop is disabled: system.kiosk_stop_command is not configured", a.systemService.StopKiosk)
}

// systemAllRestartHandler runs the configured system.all_restart_command.
//
//	@Summary		Restart server and kiosk
//	@Description	Runs the configured system.all_restart_command. Disabled (400) when none is configured (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/all/restart [post]
func (a *API) systemAllRestartHandler(w http.ResponseWriter, r *http.Request) {
	a.runSystemCommand(w, r, a.systemService.AllRestartEnabled(), "restarting both is disabled: system.all_restart_command is not configured", a.systemService.RestartAll)
}

// systemAllStopHandler runs the configured system.all_stop_command.
//
//	@Summary		Stop server and kiosk
//	@Description	Runs the configured system.all_stop_command. Disabled (400) when none is configured (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/all/stop [post]
func (a *API) systemAllStopHandler(w http.ResponseWriter, r *http.Request) {
	a.runSystemCommand(w, r, a.systemService.AllStopEnabled(), "stopping both is disabled: system.all_stop_command is not configured", a.systemService.StopAll)
}

// runSystemCommand runs a configured system command and answers with 204,
// or with 400 and the disabled message if the command is not configured.
func (a *API) runSystemCommand(w http.ResponseWriter, r *http.Request, enabled bool, disabled string, run func() error) {
	if !enabled {
		a.badRequestResponse(w, r, errors.New(disabled))
		return
	}
	if err := run(); err != nil {
		a.respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// systemProcessExitHandler exits the running application process directly.
//
//	@Summary		Exit process
//	@Description	Gracefully exits the running application process directly. Whether it comes back up depends on its supervisor, such as the restart policy of a service (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/process/exit [post]
func (a *API) systemProcessExitHandler(w http.ResponseWriter, _ *http.Request) {
	// Written before ExitProcess so the response reaches the client:
	// ExitProcess cancels the root context, which triggers http.Server.Shutdown,
	// which waits for this handler to return before tearing down connections.
	w.WriteHeader(http.StatusNoContent)
	a.systemService.ExitProcess()
}

// systemHostRebootHandler reboots the host machine.
//
//	@Summary		Reboot host
//	@Description	Reboots the host machine directly (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/host/reboot [post]
func (a *API) systemHostRebootHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.systemService.RebootHost(); err != nil {
		a.respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// systemHostShutdownHandler powers off the host machine.
//
//	@Summary		Shut down host
//	@Description	Powers off the host machine directly (Admin only).
//	@Tags			system
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/system/host/shutdown [post]
func (a *API) systemHostShutdownHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.systemService.ShutdownHost(); err != nil {
		a.respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
