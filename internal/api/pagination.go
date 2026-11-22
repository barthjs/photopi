package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

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

// writePaginated writes a paginated list response with the standard envelope shape (data + totalCount/limit/offset),
// the shared shape behind every JSON:API style list endpoint.
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

// pageParams reads the page limit and offset query parameters.
func (p *queryParser) pageParams(defaultLimit, maxLimit int) (int, int) {
	if p.err != nil {
		return 0, 0
	}
	limit, offset, err := queryPageParams(p.values, defaultLimit, maxLimit)
	p.fail(err)
	return limit, offset
}

// queryPageParams parses the JSON:API offset pagination query parameters "page[limit]" and "page[offset]".
func queryPageParams(values url.Values, defaultLimit int, maxLimit int) (int, int, error) {
	limit := defaultLimit
	if raw := strings.TrimSpace(values.Get("page[limit]")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return 0, 0, errors.New("invalid query parameter page[limit]")
		}
		limit = parsed
	}

	limit = min(limit, maxLimit)

	offset := 0
	if raw := strings.TrimSpace(values.Get("page[offset]")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return 0, 0, errors.New("invalid query parameter page[offset]")
		}
		offset = parsed
	}

	return limit, offset, nil
}
