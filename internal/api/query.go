package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// maxListLength bounds the number of values of a comma-separated query parameter.
const maxListLength = 200

// queryParser reads query parameters while remembering the first error.
// Once an error is recorded, later reads return zero values.
type queryParser struct {
	values url.Values
	err    error
}

// newQueryParser builds a queryParser over a request's query string.
func newQueryParser(r *http.Request) *queryParser {
	return &queryParser{
		values: r.URL.Query(),
	}
}

// optionalString reads an optional text query parameter.
func (p *queryParser) optionalString(key string) *string {
	return p.optional(key, parseText)
}

// optionalInt reads an optional integer query parameter.
func (p *queryParser) optionalInt(key string) *int {
	return p.optional(key, strconv.Atoi)
}

// optionalBool reads an optional boolean query parameter.
func (p *queryParser) optionalBool(key string) *bool {
	return p.optional(key, parseBool)
}

// optionalList reads an optional comma-separated query parameter, each value through the parse function.
func (p *queryParser) optionalList[T any](key string, parse func(raw string) (T, error)) []T {
	values := p.optional(key, func(raw string) ([]T, error) {
		return parseList(raw, parse)
	})
	if values == nil {
		return nil
	}
	return *values
}

// optional reads an optional query parameter through the parse function, nil if it is missing or blank.
// A value the parse function rejects records an error that names the parameter.
func (p *queryParser) optional[T any](key string, parse func(raw string) (T, error)) *T {
	if p.err != nil {
		return nil
	}

	raw := strings.TrimSpace(p.values.Get(key))
	if raw == "" {
		return nil
	}

	value, err := parse(raw)
	if err != nil {
		p.fail(fmt.Errorf("invalid query parameter %s", key))
		return nil
	}
	return &value
}

// fail records err as the first error, unless one is already set.
func (p *queryParser) fail(err error) {
	if p.err == nil {
		p.err = err
	}
}

// pathInt parses a positive integer path parameter. A missing or malformed value yields an error for a 400 response.
func pathInt(r *http.Request, name string) (int, error) {
	raw := pathParam(r, name)
	if raw == "" {
		return 0, fmt.Errorf("missing path parameter %s", name)
	}
	value, err := parsePositiveInt(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid path parameter %s", name)
	}
	return value, nil
}

// pathParam returns a path parameter decoded. Chi keeps a parameter escaped whenever the request escapes the path
// differently than Go would, as a browser does with parentheses.
func pathParam(r *http.Request, name string) string {
	raw := chi.URLParam(r, name)
	value, err := url.PathUnescape(raw)
	if err != nil {
		// A malformed escape names no stored resource, so the raw value leads to a not found answer
		return raw
	}
	return value
}

// parsePositiveInt parses a positive integer.
func parsePositiveInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid positive integer %q", raw)
	}
	return value, nil
}

// parseText returns a text query parameter as it is.
func parseText(raw string) (string, error) {
	return raw, nil
}

// parseBool parses a boolean query parameter, which also accepts 1 and 0 and yes and no regardless of case.
func parseBool(raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", raw)
	}
}

// parseList parses a comma-separated list, each value through the parse function, at most maxListLength of them.
func parseList[T any](raw string, parse func(raw string) (T, error)) ([]T, error) {
	parts := strings.Split(raw, ",")
	if len(parts) > maxListLength {
		return nil, fmt.Errorf("more than %d values", maxListLength)
	}
	values := make([]T, 0, len(parts))
	for _, part := range parts {
		value, err := parse(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
