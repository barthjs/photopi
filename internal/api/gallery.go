package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/service"
)

const (
	galleryDefaultPageLimit = 24
	galleryMaxPageLimit     = 100
	// galleryImageCacheControl lets the browser keep the images of the gallery but ask on every use whether they changed,
	// since a retaken capture replaces the image at the same address.
	galleryImageCacheControl = "private, no-cache"
	// takenAtLayout formats the time a run was taken as the wall time of the booth, which knows no time zone.
	takenAtLayout = "2006-01-02T15:04:05"
	// eventDownloadTimeout bounds the download of every photo of an event or of many runs, which can be several gigabytes.
	eventDownloadTimeout = time.Hour
)

// galleryListHandler lists the runs of an event, paginated, most recent first.
//
//	@Summary		List gallery
//	@Description	Lists the stored runs of an event, or of every event without one, paginated and ordered by the time they were taken, most recent first. Runs without a known time come last (Admin only).
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

type GalleryRunResponse struct {
	Run           int    `json:"run"`
	Event         string `json:"event"`
	ImageCount    int    `json:"imageCount"`
	OriginalCount int    `json:"originalCount"`
	// TakenAt is the time of the earliest capture as the wall time of the booth without a time zone,
	// formatted as YYYY-MM-DDTHH:MM:SS and null if unknown.
	TakenAt *string `json:"takenAt"`
}

// toGalleryRunResponse maps a run summary to its response.
func toGalleryRunResponse(r service.RunSummary) GalleryRunResponse {
	var takenAt *string
	if !r.TakenAt.IsZero() {
		formatted := r.TakenAt.Format(takenAtLayout)
		takenAt = &formatted
	}

	return GalleryRunResponse{
		Run:           r.Run,
		Event:         r.Event,
		ImageCount:    r.ImageCount,
		OriginalCount: r.OriginalCount,
		TakenAt:       takenAt,
	}
}

// galleryDeleteHandler deletes every stored file of a run.
//
//	@Summary		Delete gallery run
//	@Description	Deletes every stored file of a run, its captures, originals, thumbnail, and collage pages (Admin only).
//	@Tags			gallery
//	@Param			run	path	int	true	"Run number"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
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
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type GalleryBulkRequest struct {
	Action string `json:"action" validate:"required,oneof=delete"`
	bulkIDs
}

// galleryBulkHandler performs a bulk action on runs.
//
//	@Summary		Apply bulk action to gallery
//	@Description	Performs an action on multiple runs, given by their numbers. An unknown or malformed run fails (Admin only).
//	@Tags			gallery
//	@Accept			json
//	@Produce		json
//	@Param			body	body		GalleryBulkRequest	true	"Bulk action details"
//	@Success		200		{object}	BulkResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/bulk [post]
func (a *API) galleryBulkHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[GalleryBulkRequest](w, r)
	if !ok {
		return
	}

	result := BulkResponse{
		FailedIDs: make([]string, 0),
	}
	for _, rawID := range req.IDs {
		run, parseErr := strconv.Atoi(rawID)
		if parseErr != nil || run <= 0 {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, rawID)
			continue
		}

		var err error
		switch req.Action {
		case "delete":
			err = a.galleryService.Delete(r.Context(), run)
		}

		if err != nil {
			if !errors.Is(err, apperr.ErrNotFound) {
				logRequestFields(a.logger.Error().Err(err), r).Str("run", rawID).Str("action", req.Action).Msg("bulk action failed for run")
			}
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, rawID)
			continue
		}

		result.Succeeded++
	}

	a.writeJSON(w, r, http.StatusOK, &result, nil)
}

// galleryZipHandler streams every stored capture of the chosen runs, or their originals, as one zip archive.
//
//	@Summary		Download gallery
//	@Description	Streams every stored capture of the chosen runs as one zip archive, or their originals without the overlay. Runs without such a file are left out (Admin only).
//	@Tags			gallery
//	@Produce		application/zip
//	@Param			filter[run]	query	string	true	"Comma-separated run numbers (max 200)"
//	@Param			originals	query	bool	false	"Download the originals without the overlay"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/zip [get]
func (a *API) galleryZipHandler(w http.ResponseWriter, r *http.Request) {
	p := newQueryParser(r)
	runs := p.optionalList("filter[run]", parsePositiveInt)
	rawOriginals := p.optionalBool("originals")
	if p.err != nil {
		a.badRequestResponse(w, r, p.err)
		return
	}
	if len(runs) == 0 {
		a.badRequestResponse(w, r, errors.New("missing query parameter filter[run]"))
		return
	}
	originals := rawOriginals != nil && *rawOriginals

	files, err := a.galleryService.ListFiles(r.Context(), runs, originals)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	// A selection of a single event names the archive after it
	name := "runs"
	if event := files[0].Event; !slices.ContainsFunc(files, func(file domain.CaptureFile) bool { return file.Event != event }) {
		name = event + "-runs"
	}

	a.writeGalleryZip(w, r, files, name, originals)
}

// galleryRunZipHandler streams every stored capture of a run, or its originals, as a zip archive.
//
//	@Summary		Download gallery run
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
func (a *API) galleryRunZipHandler(w http.ResponseWriter, r *http.Request) {
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

	files, err := a.galleryService.ListFiles(r.Context(), []int{run}, originals)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeGalleryZip(w, r, files, fmt.Sprintf("%s-run-%06d", files[0].Event, run), originals)
}

// galleryDayListHandler counts the runs per day they were taken on, most recent first.
//
//	@Summary		List gallery days
//	@Description	Counts the stored runs of an event, or of every event without one, per calendar day in the time zone of the server, most recent first. Runs without a known time are left out (Admin only).
//	@Tags			gallery
//	@Produce		json
//	@Param			filter[event]	query		string	false	"Slug of the event"
//	@Success		200				{object}	GalleryDayListResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/days [get]
func (a *API) galleryDayListHandler(w http.ResponseWriter, r *http.Request) {
	p := newQueryParser(r)
	event := p.optionalString("filter[event]")
	if p.err != nil {
		a.badRequestResponse(w, r, p.err)
		return
	}

	days, err := a.galleryService.ListDays(r.Context(), event)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	response := GalleryDayListResponse{
		Days: make([]GalleryDayResponse, 0, len(days)),
	}
	for _, day := range days {
		response.Days = append(response.Days, GalleryDayResponse{
			Day:   day.Day.Format(time.DateOnly),
			Count: day.RunCount,
		})
	}

	a.writeJSON(w, r, http.StatusOK, response, nil)
}

type GalleryDayListResponse struct {
	// Days are the days with stored runs, the most recent first.
	Days []GalleryDayResponse `json:"days"`
}

type GalleryDayResponse struct {
	// Day is the date in the time zone of the server, formatted as YYYY-MM-DD.
	Day   string `json:"day"`
	Count int    `json:"count"`
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
		a.respondError(w, r, err)
		return
	}

	response := make([]GalleryEventResponse, 0, len(events))
	for _, event := range events {
		response = append(response, GalleryEventResponse{
			Event:         event.Event,
			Current:       event.Current,
			ImageCount:    event.ImageCount,
			OriginalCount: event.OriginalCount,
		})
	}

	a.writeJSON(w, r, http.StatusOK, response, nil)
}

type GalleryEventResponse struct {
	Event string `json:"event"`
	// Current marks the event the booth files new captures under.
	Current       bool `json:"current"`
	ImageCount    int  `json:"imageCount"`
	OriginalCount int  `json:"originalCount"`
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
	event := pathParam(r, "event")
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

	files, err := a.galleryService.ListEventFiles(r.Context(), event, originals)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeGalleryZip(w, r, files, event, originals)
}

// writeGalleryZip streams files as a zip archive named after name, marked as originals if they are.
// The caller lists the files before, so an unknown run or event still gets a proper 404.
func (a *API) writeGalleryZip(w http.ResponseWriter, r *http.Request, files []domain.CaptureFile, name string, originals bool) {
	if originals {
		name += "-originals"
	}
	download := &downloadWriter{
		writer:      w,
		contentType: "application/zip",
		filename:    name + ".zip",
	}
	err := a.galleryService.Zip(r.Context(), download, files)
	if err == nil {
		return
	}
	if !download.started {
		a.respondError(w, r, err)
		return
	}
	// The headers and part of the archive are already sent, so the client may only see a broken download
	logRequestFields(a.logger.Warn().Err(err), r).Msg("streaming the gallery archive failed")
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
//	@Success		304
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/{run}/captures/{index} [get]
func (a *API) galleryCaptureGetHandler(w http.ResponseWriter, r *http.Request) {
	a.serveGalleryCapture(w, r, false)
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
//	@Success		304
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/{run}/originals/{index} [get]
func (a *API) galleryOriginalGetHandler(w http.ResponseWriter, r *http.Request) {
	a.serveGalleryCapture(w, r, true)
}

// serveGalleryCapture serves the capture, or the original, of the {run} and {index} path parameters.
func (a *API) serveGalleryCapture(w http.ResponseWriter, r *http.Request, original bool) {
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

	file, err := a.galleryService.GetCapture(r.Context(), run, index, original)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.serveGalleryFile(w, r, file)
}

// galleryThumbnailGetHandler serves the thumbnail of a run as JPEG.
//
//	@Summary		Get gallery thumbnail
//	@Description	Serves the thumbnail of a run as JPEG, the screen version of its first capture. A run stored without a thumbnail serves its first capture instead (Admin only).
//	@Tags			gallery
//	@Produce		image/jpeg
//	@Param			run	path	int	true	"Run number"
//	@Success		200
//	@Success		304
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/gallery/{run}/thumbnail [get]
func (a *API) galleryThumbnailGetHandler(w http.ResponseWriter, r *http.Request) {
	run, err := pathInt(r, "run")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	file, err := a.galleryService.GetThumbnail(r.Context(), run)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.serveGalleryFile(w, r, file)
}

// serveGalleryFile streams an image of the gallery, or answers 304 if the browser holds it already.
// The storage path names the file with its run, image index, and time, so it serves as the entity tag.
func (a *API) serveGalleryFile(w http.ResponseWriter, r *http.Request, file domain.CaptureFile) {
	etag := fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(file.Path)))
	w.Header().Set("Cache-Control", galleryImageCacheControl)
	w.Header().Set("ETag", etag)
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	reader, err := a.galleryService.OpenFile(r.Context(), file)
	if err != nil {
		a.respondError(w, r, err)
		return
	}
	defer func() { _ = reader.Close() }()

	w.Header().Set("Content-Type", "image/jpeg")
	if _, err := io.Copy(w, reader); err != nil {
		// The status is sent already, so the client only sees a broken image
		logRequestFields(a.logger.Warn().Err(err), r).Msg("streaming gallery image failed")
	}
}

// etagMatches reports whether the If-None-Match header names the entity tag, compared weakly as RFC 9110 asks.
func etagMatches(header, etag string) bool {
	for candidate := range strings.SplitSeq(header, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}
