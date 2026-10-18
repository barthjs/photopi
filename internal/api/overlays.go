package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/barthjs/photopi/internal/domain"
)

const (
	maxOverlayUploadMB = 10
)

var (
	allowedOverlayUploadTypes = []string{"image/png"}
)

// overlayListHandler lists every stored overlay PNG.
//
//	@Summary		List overlays
//	@Description	Lists every stored overlay PNG (Admin only).
//	@Tags			camera
//	@Produce		json
//	@Success		200	{array}		OverlayResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/overlays	[get]
func (a *API) overlayListHandler(w http.ResponseWriter, r *http.Request) {
	overlays, err := a.overlayService.List(r.Context())
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	response := make([]OverlayResponse, 0, len(overlays))
	for _, o := range overlays {
		response = append(response, toOverlayResponse(o))
	}

	a.writeJSON(w, r, http.StatusOK, response, nil)
}

// OverlayResponse is a stored overlay image.
type OverlayResponse struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// overlayCreateHandler stores an uploaded PNG as a new overlay.
//
//	@Summary		Create overlay
//	@Description	Stores an uploaded PNG as a new overlay (Admin only).
//	@Tags			camera
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		201	{object}	OverlayResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/overlays	[post]
func (a *API) overlayCreateHandler(w http.ResponseWriter, r *http.Request) {
	data, filename, ok := a.readUpload(w, r, maxOverlayUploadMB, allowedOverlayUploadTypes)
	if !ok {
		return
	}

	overlay, err := a.overlayService.Create(r.Context(), filename, data)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusCreated, toOverlayResponse(overlay), nil)
}

// toOverlayResponse maps an overlay to its response.
func toOverlayResponse(o domain.Overlay) OverlayResponse {
	return OverlayResponse{
		Name: o.Name,
		Path: o.Path,
	}
}

// overlayDeleteHandler removes a stored overlay by name.
//
//	@Summary		Delete overlay
//	@Description	Removes a stored overlay by name. The overlay drawn over the captures cannot be deleted (409) (Admin only).
//	@Tags			camera
//	@Param			name	path	string	true	"Overlay filename"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/overlays/{name}	[delete]
func (a *API) overlayDeleteHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := a.overlayService.Delete(r.Context(), name); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// overlayDownloadHandler downloads a stored overlay as it is stored.
//
//	@Summary		Download overlay
//	@Description	Downloads a stored overlay PNG by name (Admin only).
//	@Tags			camera
//	@Produce		image/png
//	@Param			name	path	string	true	"Overlay filename"
//	@Success		200
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/overlays/{name}	[get]
func (a *API) overlayDownloadHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	data, err := a.overlayService.Download(r.Context(), name)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	writeDownloadHeaders(w, "image/png", name)
	_, _ = w.Write(data)
}
