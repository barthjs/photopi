package api

import (
	"net/http"
)

// realtimeConnectHandler opens the WebSocket connection of a client, which receives every message of the application.
//
//	@Summary		Connect realtime
//	@Description	Upgrades the request to a WebSocket that receives JSON messages with a type and optional data. On connect, a client receives hello with the version and a hash of the settings, and health with the state of every dependency. Afterward it receives health whenever a dependency changes, such as the camera.
//	@Tags			realtime
//	@Success		101
//	@Failure		400	{string}	string	"Not a WebSocket upgrade"
//	@Router			/ws [get]
func (a *API) realtimeConnectHandler(w http.ResponseWriter, r *http.Request) {
	a.hub.Serve(w, r, nil)
}
