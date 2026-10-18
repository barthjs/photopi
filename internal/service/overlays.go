package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/store"
)

// overlayExtension is the extension a stored overlay ends with, matching the PNG files the upload accepts.
const overlayExtension = ".png"

// ErrOverlayActive is returned when the overlay to delete is the one drawn over the captures.
var ErrOverlayActive = errors.New("service: the overlay is in use")

// OverlayOptions holds the settings of an OverlayService.
type OverlayOptions struct {
	// ActivePath is the storage path of the overlay drawn over the captures, empty for none.
	ActivePath string
}

// OverlayService manages the overlay images.
type OverlayService struct {
	options OverlayOptions
	store   *store.OverlayStore
}

// NewOverlayService creates an OverlayService.
func NewOverlayService(options OverlayOptions, store *store.OverlayStore) *OverlayService {
	return &OverlayService{
		options: options,
		store:   store,
	}
}

// List returns all overlays, sorted by name.
func (s *OverlayService) List(ctx context.Context) ([]domain.Overlay, error) {
	return s.store.List(ctx)
}

// Get fetches the raw bytes of the overlay at a storage path, or apperr.ErrNotFound if it does not exist.
func (s *OverlayService) Get(ctx context.Context, path string) ([]byte, error) {
	return s.store.Get(ctx, path)
}

// Create stores data as an overlay under the sanitized filename of the upload with the .png extension,
// replacing an overlay of the same name. A name that cannot be stored fails with a validation error of the file.
func (s *OverlayService) Create(ctx context.Context, filename string, data []byte) (domain.Overlay, error) {
	name := sanitizeFilename(filename)
	if strings.EqualFold(path.Ext(name), overlayExtension) {
		name = name[:len(name)-len(overlayExtension)]
	}
	if name == "" || name == "." || name == ".." {
		name = fmt.Sprintf("overlay-%d", time.Now().UnixNano())
	}

	overlay, err := s.store.Create(ctx, name+overlayExtension, data)
	if errors.Is(err, store.ErrInvalidName) {
		// A name that is still too long after sanitizing cannot be stored
		return domain.Overlay{}, apperr.NewValidationError("file", "invalid")
	}
	return overlay, err
}

// Delete removes the overlay with the given name, or returns apperr.ErrNotFound if it does not exist
// or name cannot address an overlay file. The active overlay fails with ErrOverlayActive.
func (s *OverlayService) Delete(ctx context.Context, name string) error {
	overlayPath, err := s.store.Path(name)
	if errors.Is(err, store.ErrInvalidName) {
		return apperr.ErrNotFound
	}
	if err != nil {
		return err
	}

	if overlayPath == path.Clean(s.options.ActivePath) {
		return ErrOverlayActive
	}

	return s.store.Delete(ctx, name)
}

// Download returns the content of the overlay with the given name, or apperr.ErrNotFound if it does not exist
// or name cannot address an overlay file.
func (s *OverlayService) Download(ctx context.Context, name string) ([]byte, error) {
	overlayPath, err := s.store.Path(name)
	if errors.Is(err, store.ErrInvalidName) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return s.store.Get(ctx, overlayPath)
}

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// sanitizeFilename strips path separators and unsafe characters from a client-supplied name.
// A name that leaves nothing usable sanitizes to "", so callers can fall back or reject it.
func sanitizeFilename(name string) string {
	name = path.Base(strings.TrimSpace(name))
	if name == "." || name == ".." || name == "/" {
		return ""
	}
	return unsafeFilenameChars.ReplaceAllString(name, "-")
}
