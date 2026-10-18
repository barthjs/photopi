package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
)

const (
	// overlaysFolder is the storage folder the overlays are kept in.
	overlaysFolder = "overlays"
	// overlayExtension is the extension of an overlay file, which is part of the name of the overlay.
	overlayExtension = ".png"
)

// OverlayStore keeps the overlay images as PNG files in the storage, named after the overlay.
type OverlayStore struct {
	storage media.Storage

	mu sync.Mutex
}

// NewOverlayStore creates an OverlayStore.
func NewOverlayStore(storage media.Storage) *OverlayStore {
	return &OverlayStore{
		storage: storage,
	}
}

// List returns all overlays, sorted by name.
func (s *OverlayStore) List(ctx context.Context) ([]domain.Overlay, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	paths, err := s.storage.List(ctx, overlaysFolder+"/")
	if err != nil {
		return nil, fmt.Errorf("store: listing overlays: %w", err)
	}

	overlays := make([]domain.Overlay, 0, len(paths))
	for _, p := range paths {
		name, ok := fileName(overlaysFolder, p, overlayExtension)
		if !ok {
			continue
		}
		overlays = append(overlays, domain.Overlay{
			Name: name + overlayExtension,
			Path: p,
		})
	}
	slices.SortFunc(overlays, func(a, b domain.Overlay) int { return strings.Compare(a.Name, b.Name) })

	return overlays, nil
}

// Get returns the content of the overlay at a storage path, or apperr.ErrNotFound if it does not exist.
// It takes the path, since the configuration names the active overlay by its path.
func (s *OverlayStore) Get(ctx context.Context, p string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := media.ReadAll(ctx, s.storage, p)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading overlay %q: %w", p, err)
	}

	return data, nil
}

// Create writes data as the overlay with the given name, replacing an existing one.
// A name that cannot address an overlay file fails with ErrInvalidName.
func (s *OverlayStore) Create(ctx context.Context, name string, data []byte) (domain.Overlay, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.Path(name)
	if err != nil {
		return domain.Overlay{}, err
	}

	if err := s.storage.Put(ctx, p, bytes.NewReader(data), int64(len(data))); err != nil {
		return domain.Overlay{}, fmt.Errorf("store: saving overlay %q: %w", name, err)
	}

	return domain.Overlay{
		Name: name,
		Path: p,
	}, nil
}

// Delete removes the overlay with the given name, or returns apperr.ErrNotFound if it does not exist.
// A name that cannot address an overlay file fails with ErrInvalidName.
func (s *OverlayStore) Delete(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.Path(name)
	if err != nil {
		return err
	}

	exists, err := s.storage.Exists(ctx, p)
	if err != nil {
		return fmt.Errorf("store: checking overlay %q: %w", name, err)
	}
	if !exists {
		return apperr.ErrNotFound
	}

	if err := s.storage.Delete(ctx, p); err != nil {
		return fmt.Errorf("store: deleting overlay %q: %w", name, err)
	}

	return nil
}

// Path returns the storage path of the overlay with the given name, or ErrInvalidName for a name
// without the overlay extension or one that cannot address an overlay file.
func (s *OverlayStore) Path(name string) (string, error) {
	base, found := strings.CutSuffix(name, overlayExtension)
	if !found {
		return "", ErrInvalidName
	}
	return filePath(overlaysFolder, base, overlayExtension)
}
