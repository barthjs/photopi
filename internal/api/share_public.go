package api

import (
	"fmt"
	"net/http"
)

// publicShareGetHandler lists the files of a shared run as signed URLs.
//
//	@Summary		Get public share
//	@Description	Lists the captures and collage pages of a shared run as signed URLs.
//	@Tags			share
//	@Produce		json
//	@Param			token	path		string	true	"Signed share token"
//	@Success		200		{object}	PublicShareResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/share/public/{token} [get]
func (a *API) publicShareGetHandler(w http.ResponseWriter, r *http.Request) {
	run, ok := a.resolvePublicShareRun(w, r)
	if !ok {
		return
	}

	urls, err := a.shareService.ListSharedFiles(r.Context(), run)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	a.writeJSON(w, r, http.StatusOK, PublicShareResponse{
		ImageURLs: urls,
	}, nil)
}

type PublicShareResponse struct {
	// ImageURLs are signed URLs of the captures followed by the collage pages, valid for a limited time.
	ImageURLs []string `json:"imageUrls"`
}

// publicShareZipHandler streams every file of a shared run as a zip archive.
//
//	@Summary		Download public share
//	@Description	Streams the captures and collage pages of a shared run as a zip archive.
//	@Tags			share
//	@Produce		application/zip
//	@Param			token	path	string	true	"Signed share token"
//	@Success		200
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/share/public/{token}/zip [get]
func (a *API) publicShareZipHandler(w http.ResponseWriter, r *http.Request) {
	run, ok := a.resolvePublicShareRun(w, r)
	if !ok {
		return
	}

	// The files are resolved before the headers, so a deleted run still gets a proper 404
	paths, err := a.shareService.ListSharedFilePaths(r.Context(), run)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	download := &downloadWriter{
		writer:      w,
		contentType: "application/zip",
		filename:    fmt.Sprintf("run-%06d.zip", run),
	}
	err = a.shareService.ZipSharedFiles(r.Context(), download, paths)
	if err == nil {
		return
	}
	if !download.started {
		a.respondError(w, r, err)
		return
	}
	// The headers and part of the archive are already sent, so the client may only see a broken download
	logRequestFields(a.logger.Warn().Err(err), r).Msg("streaming the shared archive failed")
}

// resolvePublicShareRun verifies the {token} URL parameter and returns the run number it was signed for.
// On failure, it writes a 404 itself and returns false:
// a bad signature, an expired link, and an unknown run are indistinguishable to the caller,
// so forged tokens cannot be used to probe which run numbers exist.
func (a *API) resolvePublicShareRun(w http.ResponseWriter, r *http.Request) (int, bool) {
	run, ok := a.shareService.ResolveSelfShareToken(pathParam(r, "token"), a.now())
	if !ok {
		a.notFoundResponse(w, r)
		return 0, false
	}

	return run, true
}
