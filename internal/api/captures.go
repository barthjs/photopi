package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/service"
)

const maxCaptureUploadMB = 25

var allowedCaptureUploadTypes = []string{"image/jpeg"}

// captureUploadRules are the format and the resolution bounds of an uploaded capture.
var captureUploadRules = imageRules{
	Format:  media.ImageFormatJPEG,
	MaxEdge: service.MaxCaptureEdge,
}

// captureGetHandler serves one stored capture of a run as JPEG.
//
//	@Summary		Get capture
//	@Description	Serves one stored capture of a run as JPEG.
//	@Tags			camera
//	@Produce		image/jpeg
//	@Param			run		path	int	true	"Run number"
//	@Param			index	path	int	true	"Image index within the run (1-based)"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/camera/runs/{run}/captures/{index} [get]
func (a *API) captureGetHandler(w http.ResponseWriter, r *http.Request) {
	run := sessionRun(r)

	index, err := pathInt(r, "index")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	data, err := a.captureService.Get(r.Context(), run, index)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "image/jpeg")
	_, _ = w.Write(data)
}

// captureCreateHandler takes one still and stores it as the next image of the currently open run.
//
//	@Summary		Create capture
//	@Description	Captures a still (backend camera sources) or accepts an uploaded still (webcam source) and stores it as the next image of run. Fails with 409 if run is not the currently open run.
//	@Tags			camera
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			run	path		int	true	"Run number"
//	@Success		201	{object}	CaptureResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Failure		502	{object}	ErrorResponse
//	@Router			/camera/runs/{run}/captures [post]
func (a *API) captureCreateHandler(w http.ResponseWriter, r *http.Request) {
	run := sessionRun(r)

	var data *[]byte
	if a.options.Camera.Source == camera.SourceWebcam {
		uploaded, _, ok := a.readImageUpload(w, r, maxCaptureUploadMB, allowedCaptureUploadTypes, captureUploadRules)
		if !ok {
			return
		}
		data = &uploaded
	} else {
		var err error
		data, err = a.camera.Capture(r.Context())
		if err != nil {
			// A client that left canceled the capture, which says nothing about the camera
			if errors.Is(r.Context().Err(), context.Canceled) {
				logRequestFields(a.logger.Info().Err(err), r).Msg("camera capture canceled by the client")
				return
			}
			a.cameraHealth.RecordFailure(err)
			a.serverErrorResponse(w, r, err)
			return
		}
		a.cameraHealth.RecordSuccess()
	}

	capture, err := a.captureService.Create(r.Context(), run, data)
	if err != nil {
		if errors.Is(err, service.ErrCaptureCorrupt) {
			// The camera reported success but delivered an unusable image, such as a truncated one.
			// The camera health turns back to failed, so a guest is not led into a run the camera cannot complete.
			a.cameraHealth.RecordFailure(err)
			logRequestFields(a.logger.Warn().Err(err), r).Msg("camera returned an unusable capture")
		}
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusCreated, CaptureResponse{
		Run:          capture.Run,
		ImageIndex:   capture.ImageIndex,
		ImagesPerRun: capture.ImagesPerRun,
	}, nil)
}

type CaptureResponse struct {
	Run int `json:"run"`
	// ImageIndex is the position of the capture in the run, which is complete at ImagesPerRun.
	ImageIndex   int `json:"imageIndex"`
	ImagesPerRun int `json:"imagesPerRun"`
}

// capturePreviewGetHandler serves the screen version of one capture of a run as JPEG,
// small enough for the kiosk to show right after the shot and on the review screen.
//
//	@Summary		Get capture preview
//	@Description	Serves the screen version of one capture of a run as JPEG, with the overlay drawn in.
//	@Tags			camera
//	@Produce		image/jpeg
//	@Param			run		path	int	true	"Run number"
//	@Param			index	path	int	true	"Image index within the run (1-based)"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/camera/runs/{run}/captures/{index}/preview [get]
func (a *API) capturePreviewGetHandler(w http.ResponseWriter, r *http.Request) {
	run := sessionRun(r)

	index, err := pathInt(r, "index")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	data, err := a.captureService.GetPreview(r.Context(), run, index)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "image/jpeg")
	_, _ = w.Write(data)
}
