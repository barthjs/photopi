package api

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"maps"
	"mime"
	"net/http"
)

// PaginatedMeta holds the metadata for paginated responses.
type PaginatedMeta struct {
	TotalCount int `json:"totalCount"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
}

// PaginatedResponse wraps a slice of data with pagination metadata.
type PaginatedResponse[T any] struct {
	Data []T           `json:"data"`
	Meta PaginatedMeta `json:"meta"`
}

// writePaginated writes a paginated list response with the standard envelope shape
// (data + totalCount/limit/offset), the shared shape behind every JSON:API style list endpoint.
func (a *API) writePaginated[T any](w http.ResponseWriter, r *http.Request, data []T, totalCount, limit, offset int) {
	a.writeJSON(w, r, http.StatusOK, PaginatedResponse[T]{
		Data: data,
		Meta: PaginatedMeta{
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
		},
	}, nil)
}

// writeJSON writes a JSON response and, if encoding fails, reports it as a server error.
func (a *API) writeJSON(w http.ResponseWriter, r *http.Request, status int, data any, headers http.Header) {
	if err := writeJSON(w, status, data, headers); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// writeJSON writes a JSON response with status code and optional headers.
// Durations are written as nanoseconds, as encoding/json/v2 has no default for them.
func writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
	js, err := json.Marshal(data, jsonv1.FormatDurationAsNano(true))
	if err != nil {
		return err
	}

	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(js)

	return nil
}

// writeText writes a plain text with status code and optional headers.
func writeText(w http.ResponseWriter, status int, data string, headers http.Header) {
	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(data))
}

// writeDownloadHeaders marks the response as a file download with the given content type and file name.
// The file name is quoted and encoded as the header requires, so any stored name is safe.
func writeDownloadHeaders(w http.ResponseWriter, contentType, filename string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": filename,
	}))
	w.Header().Set("Cache-Control", "no-store")
}
