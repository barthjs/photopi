package api

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/barthjs/photopi/internal/media"
)

const (
	// multipartFramingHeadroom is added to the file size limit to allow for the multipart framing.
	multipartFramingHeadroom = 256 << 10
	// uploadTimeout gives uploads more time than the default request timeout, for slow connections.
	uploadTimeout = 5 * time.Minute
)

// imageRules are the format and the resolution bounds an uploaded image must meet.
type imageRules struct {
	// Format is the only format the image may have.
	Format media.ImageFormat
	// MaxEdge bounds the pixel count of the image to a square with this edge.
	MaxEdge int
	// MinEdge is the shortest length both edges of the image must reach, 0 accepts any size.
	MinEdge int
}

// readImageUpload returns the content and name of an uploaded image whose header meets rules,
// or writes the failure and returns false.
func (a *API) readImageUpload(w http.ResponseWriter, r *http.Request, maxSizeMB int, allowedTypes []string, rules imageRules) ([]byte, string, bool) {
	data, filename, ok := a.readUpload(w, r, maxSizeMB, allowedTypes)
	if !ok {
		return nil, "", false
	}

	if err := media.ValidateImageData(
		data,
		media.WithFormats(rules.Format),
		media.WithMaxPixels(int64(rules.MaxEdge)*int64(rules.MaxEdge)),
		media.WithMinDimension(rules.MinEdge),
	); err != nil {
		switch {
		case errors.Is(err, media.ErrImageTooSmall):
			a.failedValidationResponse(w, r, FieldErrors{"file": {
				{
					Tag:   "resolution_too_low",
					Param: fmt.Sprintf("%dx%d", rules.MinEdge, rules.MinEdge),
				},
			}})
		case errors.Is(err, media.ErrImageTooLarge):
			a.failedValidationResponse(w, r, FieldErrors{"file": {
				{
					Tag:   "resolution_too_high",
					Param: fmt.Sprintf("%dx%d", rules.MaxEdge, rules.MaxEdge),
				},
			}})
		default:
			a.failedValidationResponse(w, r, FieldErrors{"file": {
				{
					Tag:   "invalid_format",
					Param: string(rules.Format),
				},
			}})
		}
		return nil, "", false
	}

	return data, filename, true
}

// readUpload returns the content and name of the only file of the multipart "file" field within maxSizeMB
// and of an allowed type, or writes the failure and returns false.
func (a *API) readUpload(w http.ResponseWriter, r *http.Request, maxSizeMB int, allowedTypes []string) ([]byte, string, bool) {
	maxFile := int64(maxSizeMB) << 20
	maxRequest := maxFile + multipartFramingHeadroom

	r.Body = http.MaxBytesReader(w, r.Body, maxRequest)
	if !a.parseMultipartForm(w, r, maxRequest, "file") {
		return nil, "", false
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		a.failedValidationResponse(w, r, FieldErrors{"file": {
			{
				Tag: "required",
			},
		}})
		return nil, "", false
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxFile+1))
	if err != nil {
		a.badRequestResponse(w, r, err)
		return nil, "", false
	}
	if int64(len(data)) > maxFile {
		a.failedValidationResponse(w, r, FieldErrors{"file": {
			{
				Tag: "too_large",
			},
		}})
		return nil, "", false
	}
	if !uploadTypeAllowed(data, allowedTypes) {
		a.failedValidationResponse(w, r, FieldErrors{"file": {
			{
				Tag: "unsupported_type",
			},
		}})
		return nil, "", false
	}

	return data, header.Filename, true
}

// parseMultipartForm parses the multipart form of a request within maxBytes, or writes the failure and returns false.
func (a *API) parseMultipartForm(w http.ResponseWriter, r *http.Request, maxBytes int64, field string) bool {
	err := r.ParseMultipartForm(maxBytes)
	if err == nil {
		return true
	}

	_, bodyTooLarge := errors.AsType[*http.MaxBytesError](err)
	if bodyTooLarge || errors.Is(err, multipart.ErrMessageTooLarge) {
		a.failedValidationResponse(w, r, FieldErrors{field: {
			{
				Tag: "too_large",
			},
		}})
		return false
	}

	a.badRequestResponse(w, r, err)
	return false
}

// uploadTypeAllowed reports whether the detected content type of data is in allowedTypes.
func uploadTypeAllowed(data []byte, allowedTypes []string) bool {
	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))
	// DetectContentType can append parameters, so only the media type is compared
	if i := strings.IndexByte(detected, ';'); i != -1 {
		detected = strings.TrimSpace(detected[:i])
	}

	for _, allowed := range allowedTypes {
		if detected == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

// uploadBodyLimit is the body limit of a route that accepts files of up to maxSizeMB in total.
func uploadBodyLimit(maxSizeMB int) int64 {
	return int64(maxSizeMB)<<20 + multipartFramingHeadroom
}
