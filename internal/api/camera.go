package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/barthjs/photopi/internal/camera"
)

// previewBoundary separates JPEG frames in the multipart/x-mixed-replace preview stream.
const previewBoundary = "photopiframe"

// cameraOverlayGetHandler serves the configured overlay PNG so the frontend can layer it over the live preview itself.
// The backend only composites the overlay onto the stored capture.
//
//	@Summary		Get camera overlay
//	@Description	Returns the configured overlay PNG, or 404 if none is configured.
//	@Tags			camera
//	@Produce		png
//	@Success		200
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/camera/overlay [get]
func (a *API) cameraOverlayGetHandler(w http.ResponseWriter, r *http.Request) {
	if a.options.Camera.OverlayPath == "" {
		a.notFoundResponse(w, r)
		return
	}

	data, err := a.overlayService.Get(r.Context(), a.options.Camera.OverlayPath)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, path.Base(a.options.Camera.OverlayPath), time.Time{}, bytes.NewReader(data))
}

// cameraPreviewGetHandler streams the live camera preview as MJPEG (multipart/x-mixed-replace),
// one JPEG frame per part, for as long as the client stays connected and keeps reading.
//
//	@Summary		Get camera preview
//	@Description	Streams the live camera preview as multipart/x-mixed-replace MJPEG.
//	@Tags			camera
//	@Produce		multipart/x-mixed-replace
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/camera/preview [get]
func (a *API) cameraPreviewGetHandler(w http.ResponseWriter, r *http.Request) {
	if a.options.Camera.Source == camera.SourceWebcam {
		a.errorResponse(w, r, http.StatusBadRequest, codeBadRequest, fmt.Sprintf("camera source %q does not provide a server-side preview stream", a.options.Camera.Source))
		return
	}

	if _, ok := w.(http.Flusher); !ok {
		a.serverErrorResponse(w, r, errors.New("camera preview: response writer does not support flushing"))
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("multipart/x-mixed-replace; boundary=%s", previewBoundary))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	client := newPreviewClient(w)
	client.Flush()

	err := a.camera.StreamPreview(r.Context(), client, previewBoundary)
	if err == nil {
		return
	}
	// A capture took over the device, the client opens a new stream afterward
	if errors.Is(err, camera.ErrPreviewStopped) {
		logRequestFields(a.logger.Debug(), r).Msg("camera preview stopped for a capture")
		return
	}
	// The client disconnected or stopped reading, which says nothing about the camera.
	// Headers are already sent, so there is nothing more to answer.
	if r.Context().Err() != nil || client.failed() {
		logRequestFields(a.logger.Debug().Err(err), r).Msg("camera preview client left")
		return
	}
	a.cameraHealth.RecordFailure(err)
	logRequestFields(a.logger.Error().Err(err), r).Msg("camera preview stream ended")
}

// previewWriteTimeout bounds writing to the preview client, so a client that stops reading ends the stream
// instead of keeping the camera busy. The route itself has no deadline, since the stream is endless.
const previewWriteTimeout = 5 * time.Second

// previewClient writes the preview stream to the client with a fresh write deadline for every write,
// and remembers whether writing to the client failed.
type previewClient struct {
	w          http.ResponseWriter
	controller *http.ResponseController
	err        error
}

// newPreviewClient creates a previewClient writing to w.
func newPreviewClient(w http.ResponseWriter) *previewClient {
	return &previewClient{
		w:          w,
		controller: http.NewResponseController(w),
	}
}

// Write writes p to the client within previewWriteTimeout.
func (c *previewClient) Write(p []byte) (int, error) {
	if err := c.extendDeadline(); err != nil {
		return 0, err
	}

	n, err := c.w.Write(p)
	if err != nil {
		c.err = err
	}
	return n, err
}

// Flush sends the buffered data to the client within previewWriteTimeout.
// A failure is remembered, and the next Write fails as well, since the connection is broken.
func (c *previewClient) Flush() {
	if err := c.extendDeadline(); err != nil {
		return
	}
	if err := c.controller.Flush(); err != nil {
		c.err = err
	}
}

// extendDeadline moves the write deadline previewWriteTimeout ahead.
// Writers without deadlines, such as test recorders, keep writing without one.
func (c *previewClient) extendDeadline() error {
	err := c.controller.SetWriteDeadline(time.Now().Add(previewWriteTimeout))
	if err != nil && !errors.Is(err, http.ErrNotSupported) {
		c.err = err
		return err
	}
	return nil
}

// failed reports whether writing to the client failed.
func (c *previewClient) failed() bool {
	return c.err != nil
}
