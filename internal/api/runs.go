package api

import (
	"net/http"

	"github.com/barthjs/photopi/internal/service"
)

type RunCreateRequest struct {
	// ScreenWidth is the width of the screen of the client in device pixels, which bounds the screen versions of the captures.
	ScreenWidth int `json:"screenWidth" validate:"required,min=1,max=16384"`
	// ScreenHeight is the height of the screen of the client in device pixels.
	ScreenHeight int `json:"screenHeight" validate:"required,min=1,max=16384"`
}

// runCreateHandler opens a new run for the session of the client, discarding an incomplete one of the same client,
// while an open run of another client blocks the start.
//
//	@Summary		Create run
//	@Description	Opens a new run and returns its number and the number of captures it expects. The screen of the client bounds the screen versions of the captures, at most 1920 pixels on the longest edge. Fails with 409 while another client's run is open.
//	@Tags			camera
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RunCreateRequest	true	"Screen of the client"
//	@Success		201		{object}	RunResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/camera/runs [post]
func (a *API) runCreateHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[RunCreateRequest](w, r)
	if !ok {
		return
	}

	owner, err := a.runSessionService.Owner(r.Context())
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	start, err := a.captureService.CreateRun(r.Context(), service.RunCreateInput{
		Owner:        owner,
		ScreenWidth:  req.ScreenWidth,
		ScreenHeight: req.ScreenHeight,
	})
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
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
