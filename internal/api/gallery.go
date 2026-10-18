package api

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/service"
)

const (
	galleryDefaultPageLimit = 24
	galleryMaxPageLimit     = 100

	// eventDownloadTimeout bounds the download of every photo of an event, which can be several gigabytes.
	eventDownloadTimeout = time.Hour
)

// galleryListHandler lists the runs of an event, paginated, most recent first.
//
//	@Summary		List gallery
//	@Description	Lists the stored runs of an event, or of every event without one, paginated and most recent first (Admin only).
//	@Tags			gallery
//	@Produce		json
//	@Param			filter[event]	query		string	false	"Slug of the event"
//	@Param			page[limit]		query		int		false	"Page size (max 100)"			default(24)
//	@Param			page[offset]	query		int		false	"Offset into the result set"	default(0)
//	@Success		200				{object}	PaginatedResponse[GalleryRunResponse]
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery	[get]
func (a *API) galleryListHandler(w http.ResponseWriter, r *http.Request) {
	p := newQueryParser(r)
	event := p.optionalString("filter[event]")
	limit, offset := p.pageParams(galleryDefaultPageLimit, galleryMaxPageLimit)
	if p.err != nil {
		a.badRequestResponse(w, r, p.err)
		return
	}

	runs, totalCount, err := a.galleryService.List(r.Context(), domain.RunListFilter{
		Event:  event,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	data := make([]GalleryRunResponse, 0, len(runs))
	for _, run := range runs {
		data = append(data, toGalleryRunResponse(run))
	}

	a.writePaginated(w, r, data, totalCount, limit, offset)
}

// GalleryRunResponse is a stored run with its event and its capture counts.
type GalleryRunResponse struct {
	Run           int    `json:"run"`
	Event         string `json:"event"`
	ImageCount    int    `json:"imageCount"`
	OriginalCount int    `json:"originalCount"`
}

// toGalleryRunResponse maps a run summary to its response.
func toGalleryRunResponse(r service.RunSummary) GalleryRunResponse {
	return GalleryRunResponse{
		Run:           r.Run,
		Event:         r.Event,
		ImageCount:    r.ImageCount,
		OriginalCount: r.OriginalCount,
	}
}

// galleryDeleteHandler deletes every stored capture for a run.
//
//	@Summary		Delete gallery
//	@Description	Deletes every stored capture for a run (Admin only).
//	@Tags			gallery
//	@Param			run	path	int	true	"Run number"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/{run} [delete]
func (a *API) galleryDeleteHandler(w http.ResponseWriter, r *http.Request) {
	run, err := pathInt(r, "run")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	if err := a.galleryService.Delete(r.Context(), run); err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// galleryZipHandler streams every stored capture of a run, or its originals, as a zip archive.
//
//	@Summary		Download gallery
//	@Description	Streams every stored capture of a run as a zip archive, or its originals without the overlay (Admin only).
//	@Tags			gallery
//	@Produce		application/zip
//	@Param			run			path	int		true	"Run number"
//	@Param			originals	query	bool	false	"Download the originals without the overlay"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/{run}/zip [get]
func (a *API) galleryZipHandler(w http.ResponseWriter, r *http.Request) {
	run, err := pathInt(r, "run")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	p := newQueryParser(r)
	rawOriginals := p.optionalBool("originals")
	if p.err != nil {
		a.badRequestResponse(w, r, p.err)
		return
	}
	originals := rawOriginals != nil && *rawOriginals

	// The files are resolved before the headers, so an unknown run still gets a proper 404
	paths, err := a.galleryService.ListFiles(r.Context(), run, originals)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	writeZipHeaders(w, fmt.Sprintf("run-%06d", run), originals)

	if err := a.galleryService.Zip(r.Context(), w, paths); err != nil {
		// The headers and part of the archive may already be sent, so the client may only see a broken download
		a.serverErrorResponse(w, r, err)
		return
	}
}

// galleryEventListHandler lists the events with stored runs, most recent first.
//
//	@Summary		List gallery events
//	@Description	Lists the events with stored runs, most recent first, with the event of the booth marked as current (Admin only).
//	@Tags			gallery
//	@Produce		json
//	@Success		200	{array}		GalleryEventResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/events [get]
func (a *API) galleryEventListHandler(w http.ResponseWriter, r *http.Request) {
	events, err := a.galleryService.ListEvents(r.Context())
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	response := make([]GalleryEventResponse, 0, len(events))
	for _, event := range events {
		response = append(response, GalleryEventResponse{
			Event:         event.Event,
			Current:       event.Current,
			RunCount:      event.RunCount,
			ImageCount:    event.ImageCount,
			OriginalCount: event.OriginalCount,
		})
	}

	a.writeJSON(w, r, http.StatusOK, response, nil)
}

// GalleryEventResponse is an event with stored runs. Current marks the event the booth files new captures under.
type GalleryEventResponse struct {
	Event         string `json:"event"`
	Current       bool   `json:"current"`
	RunCount      int    `json:"runCount"`
	ImageCount    int    `json:"imageCount"`
	OriginalCount int    `json:"originalCount"`
}

// eventSlugPattern matches the slug of an event folder.
var eventSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// galleryEventZipHandler streams every stored capture of an event, or their originals, as one zip archive.
//
//	@Summary		Download gallery event
//	@Description	Streams every stored capture of an event as one zip archive, or their originals without the overlay (Admin only).
//	@Tags			gallery
//	@Produce		application/zip
//	@Param			event		path	string	true	"Slug of the event"
//	@Param			originals	query	bool	false	"Download the originals without the overlay"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/events/{event}/zip [get]
func (a *API) galleryEventZipHandler(w http.ResponseWriter, r *http.Request) {
	event := chi.URLParam(r, "event")
	if !eventSlugPattern.MatchString(event) {
		a.notFoundResponse(w, r)
		return
	}

	p := newQueryParser(r)
	rawOriginals := p.optionalBool("originals")
	if p.err != nil {
		a.badRequestResponse(w, r, p.err)
		return
	}
	originals := rawOriginals != nil && *rawOriginals

	writeZipHeaders(w, event, originals)

	if err := a.galleryService.ZipEvent(r.Context(), w, event, originals); err != nil {
		// The headers and part of the archive may already be sent, so the client may only see a broken download
		a.serverErrorResponse(w, r, err)
		return
	}
}

// galleryCaptureGetHandler serves one stored capture of a run as JPEG.
//
//	@Summary		Get gallery capture
//	@Description	Serves one stored capture of a run as JPEG (Admin only).
//	@Tags			gallery
//	@Produce		image/jpeg
//	@Param			run		path	int	true	"Run number"
//	@Param			index	path	int	true	"Image index within the run (1-based)"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/{run}/captures/{index} [get]
func (a *API) galleryCaptureGetHandler(w http.ResponseWriter, r *http.Request) {
	a.serveGalleryImage(w, r, false)
}

// galleryOriginalGetHandler serves the original of one capture of a run, stored without the overlay, as JPEG.
//
//	@Summary		Get gallery original
//	@Description	Serves the original of one capture of a run, stored without the overlay, as JPEG (Admin only).
//	@Tags			gallery
//	@Produce		image/jpeg
//	@Param			run		path	int	true	"Run number"
//	@Param			index	path	int	true	"Image index within the run (1-based)"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/{run}/originals/{index} [get]
func (a *API) galleryOriginalGetHandler(w http.ResponseWriter, r *http.Request) {
	a.serveGalleryImage(w, r, true)
}

// serveGalleryImage serves the capture of the {run} and {index} path parameters, or its original.
func (a *API) serveGalleryImage(w http.ResponseWriter, r *http.Request, original bool) {
	run, err := pathInt(r, "run")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	index, err := pathInt(r, "index")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	data, err := a.galleryService.GetCapture(r.Context(), run, index, original)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "image/jpeg")
	_, _ = w.Write(data)
}

// writeZipHeaders sets the headers of a zip download named after name, marked as originals if they are.
func writeZipHeaders(w http.ResponseWriter, name string, originals bool) {
	if originals {
		name += "-originals"
	}
	writeDownloadHeaders(w, "application/zip", name+".zip")
}
