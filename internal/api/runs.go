package api

import (
	"net/http"
)

// runCreateHandler opens a new run with a fresh run number and binds it to the session of the client.
// A run the same client left incomplete is discarded, an open run of another client blocks the start.
//
//	@Summary		Create run
//	@Description	Opens a new run and returns its number and the number of captures it expects. Fails with 409 while another client's run is open.
//	@Tags			camera
//	@Produce		json
//	@Success		201	{object}	RunResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/camera/runs [post]
func (a *API) runCreateHandler(w http.ResponseWriter, r *http.Request) {
	owner, err := a.runSessionService.Owner(r.Context())
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	start, err := a.captureService.CreateRun(r.Context(), owner)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.runSessionService.Start(r.Context(), start.Run, a.now())

	a.writeJSON(w, r, http.StatusCreated, RunResponse{
		Run:          start.Run,
		ImagesPerRun: start.ImagesPerRun,
	}, nil)
}

// RunResponse is a started run with the number of captures it takes.
type RunResponse struct {
	Run          int `json:"run"`
	ImagesPerRun int `json:"imagesPerRun"`
}

// runAbortHandler discards a run with its captures and closes it, so it can never absorb further captures.
//
//	@Summary		Abort run
//	@Description	Deletes every capture stored for a run and closes it if it is still open.
//	@Tags			camera
//	@Param			run	path	int	true	"Run number"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/camera/runs/{run} [delete]
func (a *API) runAbortHandler(w http.ResponseWriter, r *http.Request) {
	run := sessionRun(r)

	if err := a.captureService.AbortRun(r.Context(), run); err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
