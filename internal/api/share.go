package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/barthjs/photopi/internal/service"
)

const (
	// cloudUploadTimeout bounds a whole upload in the background.
	// Every request of an upload has its own, shorter limit,
	// so this only stops an upload that keeps going without getting anywhere.
	cloudUploadTimeout = 10 * time.Minute
	// emailTimeout bounds sending one share email in the background.
	emailTimeout = 3 * time.Minute
	// printTimeout bounds rendering and submitting one print in the background.
	printTimeout = 3 * time.Minute

	shareRateLimit       = 3
	shareRateLimitWindow = 30 * time.Minute
)

// shareCloudUploadGetHandler reports the current state of a run's cloud upload.
//
//	@Summary		Get cloud upload
//	@Description	Reports the current state of a run's cloud upload and its link once it is done.
//	@Tags			share
//	@Produce		json
//	@Param			run	path		int	true	"Run number"
//	@Success		200	{object}	ShareJobResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/share/{run}/cloud-upload [get]
func (a *API) shareCloudUploadGetHandler(w http.ResponseWriter, r *http.Request) {
	run := sessionRun(r)

	state, found := a.shareService.GetCloudUpload(run)
	if !found {
		a.notFoundResponse(w, r)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toShareJobResponse(state), nil)
}

type ShareJobResponse struct {
	Status string `json:"status" enums:"pending,done,failed"`
	Reason string `json:"reason,omitempty" enums:"unreachable,queue_unavailable,failed"`
	URL    string `json:"url,omitempty"`
}

// shareCloudUploadHandler starts sharing a run as a link in the background and returns immediately.
//
//	@Summary		Upload run
//	@Description	Starts sharing a run as a link in the background, uploading its captures to the configured cloud share provider first. Poll GET on the same path for the result. Only available when a cloud share is configured.
//	@Tags			share
//	@Param			run	path	int	true	"Run number"
//	@Success		202
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/share/{run}/cloud-upload [post]
func (a *API) shareCloudUploadHandler(w http.ResponseWriter, r *http.Request) {
	if !a.shareService.CloudEnabled() {
		a.badRequestResponse(w, r, errors.New("cloud sharing is disabled"))
		return
	}

	run := sessionRun(r)

	if !a.runHasCaptures(w, r, run) {
		return
	}

	if a.shareService.BeginCloudUpload(run) {
		a.background(cloudUploadTimeout, func(ctx context.Context) {
			if err := a.shareService.RunCloudUpload(ctx, run); err != nil {
				a.logger.Error().Err(err).Int("run", run).Msg("cloud upload failed")
			}
		})
	}

	w.WriteHeader(http.StatusAccepted)
}

// shareEmailGetHandler reports the current state of the latest email of a run.
//
//	@Summary		Get email
//	@Description	Reports the current state of the latest email of a run.
//	@Tags			share
//	@Produce		json
//	@Param			run	path		int	true	"Run number"
//	@Success		200	{object}	ShareJobResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/share/{run}/email [get]
func (a *API) shareEmailGetHandler(w http.ResponseWriter, r *http.Request) {
	run := sessionRun(r)

	state, found := a.shareService.GetEmail(run)
	if !found {
		a.notFoundResponse(w, r)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toShareJobResponse(state), nil)
}

type ShareEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// shareEmailHandler starts emailing every capture of a run in the background, so a slow uplink cannot run into the timeout.
//
//	@Summary		Email run
//	@Description	Starts emailing every capture of a run to the given address as attachments in the background. Poll GET on the same path for the result. Only available when email sharing is enabled.
//	@Tags			share
//	@Accept			json
//	@Param			run		path	int					true	"Run number"
//	@Param			request	body	ShareEmailRequest	true	"Recipient email address"
//	@Success		202
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/share/{run}/email [post]
func (a *API) shareEmailHandler(w http.ResponseWriter, r *http.Request) {
	if !a.shareService.EmailEnabled() {
		a.badRequestResponse(w, r, errors.New("email sharing is disabled"))
		return
	}

	run := sessionRun(r)

	req, ok := a.decodeJSON[ShareEmailRequest](w, r)
	if !ok {
		return
	}

	if !a.runHasCaptures(w, r, run) {
		return
	}

	if !a.shareService.BeginEmail(run) {
		a.conflictResponse(w, r, "an email of this run is already being sent")
		return
	}

	a.background(emailTimeout, func(ctx context.Context) {
		if err := a.shareService.RunEmail(ctx, run, req.Email); err != nil {
			a.logger.Error().Err(err).Int("run", run).Msg("share email failed")
		}
	})

	w.WriteHeader(http.StatusAccepted)
}

// sharePrintGetHandler reports the current state of the latest print of a run.
//
//	@Summary		Get print
//	@Description	Reports the current state of the latest print of a run. A failed print names its reason, queue_unavailable if the printer queue did not accept it.
//	@Tags			share
//	@Produce		json
//	@Param			run	path		int	true	"Run number"
//	@Success		200	{object}	ShareJobResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/share/{run}/print [get]
func (a *API) sharePrintGetHandler(w http.ResponseWriter, r *http.Request) {
	run := sessionRun(r)

	state, found := a.printService.Get(run)
	if !found {
		a.notFoundResponse(w, r)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toShareJobResponse(state), nil)
}

// toShareJobResponse maps the state of a share job to its response.
func toShareJobResponse(state service.JobState) ShareJobResponse {
	return ShareJobResponse{
		Status: string(state.Status),
		Reason: string(state.Reason),
		URL:    state.URL,
	}
}

// sharePrintHandler starts printing the composed print layout of a run in the background and returns immediately.
//
//	@Summary		Print run
//	@Description	Starts composing the captures of a run onto the configured print layout and submitting it to the configured CUPS printer queue in the background. Poll GET on the same path for the result. Only available when printing is enabled.
//	@Tags			share
//	@Param			run	path	int	true	"Run number"
//	@Success		202
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/share/{run}/print [post]
func (a *API) sharePrintHandler(w http.ResponseWriter, r *http.Request) {
	if !a.printService.Enabled() {
		a.badRequestResponse(w, r, errors.New("printing is disabled"))
		return
	}

	run := sessionRun(r)

	if !a.runHasCaptures(w, r, run) {
		return
	}

	if err := a.printService.CheckQueue(r.Context()); err != nil {
		if errors.Is(err, service.ErrPrintQueueUnavailable) {
			a.logger.Warn().Err(err).Int("run", run).Msg("print refused")
		}
		a.respondError(w, r, err)
		return
	}

	if a.printService.Begin(run) {
		a.background(printTimeout, func(ctx context.Context) {
			if err := a.printService.Print(ctx, run); err != nil {
				a.logger.Error().Err(err).Int("run", run).Msg("print failed")
			}
		})
	}

	w.WriteHeader(http.StatusAccepted)
}

// runHasCaptures answers 404 and returns false if no capture is stored for run.
func (a *API) runHasCaptures(w http.ResponseWriter, r *http.Request, run int) bool {
	exists, err := a.shareService.RunHasCaptures(r.Context(), run)
	if err != nil {
		a.respondError(w, r, err)
		return false
	}
	if !exists {
		a.notFoundResponse(w, r)
		return false
	}

	return true
}
