package store

import (
	"bytes"
	"context"
	"encoding/json/v2"
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
	// layoutsFolder is the storage folder the layouts are kept in.
	layoutsFolder = "layouts"
	// layoutExtension is the extension of a layout file.
	layoutExtension = ".json"
)

// LayoutStore keeps every print layout as a JSON file in the storage, named after its ID.
type LayoutStore struct {
	storage media.Storage

	mu sync.Mutex
}

// NewLayoutStore creates a LayoutStore.
func NewLayoutStore(storage media.Storage) *LayoutStore {
	return &LayoutStore{
		storage: storage,
	}
}

// List returns all layouts, sorted by ID.
func (s *LayoutStore) List(ctx context.Context) ([]domain.Layout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	paths, err := s.storage.List(ctx, layoutsFolder+"/")
	if err != nil {
		return nil, fmt.Errorf("store: listing layouts: %w", err)
	}

	layouts := make([]domain.Layout, 0, len(paths))
	for _, p := range paths {
		id, ok := fileName(layoutsFolder, p, layoutExtension)
		if !ok {
			continue
		}
		layout, err := s.read(ctx, id)
		if err != nil {
			return nil, err
		}
		layouts = append(layouts, *layout)
	}
	slices.SortFunc(layouts, func(a, b domain.Layout) int { return strings.Compare(a.ID, b.ID) })

	return layouts, nil
}

// Get returns the layout with the given ID, or apperr.ErrNotFound if it does not exist.
// An ID that cannot address a layout file fails with ErrInvalidName.
func (s *LayoutStore) Get(ctx context.Context, id string) (*domain.Layout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.read(ctx, id)
}

// read reads the layout with the given ID for List and Get, which hold the lock.
func (s *LayoutStore) read(ctx context.Context, id string) (*domain.Layout, error) {
	p, err := filePath(layoutsFolder, id, layoutExtension)
	if err != nil {
		return nil, err
	}

	data, err := media.ReadAll(ctx, s.storage, p)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading layout %q: %w", id, err)
	}

	var layout domain.Layout
	if err := json.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("store: decoding layout %q: %w", id, err)
	}

	return &layout, nil
}

// Save writes the layout under the given ID, replacing an existing one.
// An ID that cannot address a layout file fails with ErrInvalidName.
func (s *LayoutStore) Save(ctx context.Context, id string, layout domain.Layout) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := filePath(layoutsFolder, id, layoutExtension)
	if err != nil {
		return err
	}

	data, err := json.Marshal(layout)
	if err != nil {
		return fmt.Errorf("store: encoding layout %q: %w", id, err)
	}

	if err := s.storage.Put(ctx, p, bytes.NewReader(data), int64(len(data))); err != nil {
		return fmt.Errorf("store: saving layout %q: %w", id, err)
	}

	return nil
}

// Delete removes the layout with the given ID, or returns apperr.ErrNotFound if it does not exist.
// An ID that cannot address a layout file fails with ErrInvalidName.
func (s *LayoutStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := filePath(layoutsFolder, id, layoutExtension)
	if err != nil {
		return err
	}

	exists, err := s.storage.Exists(ctx, p)
	if err != nil {
		return fmt.Errorf("store: checking layout %q: %w", id, err)
	}
	if !exists {
		return apperr.ErrNotFound
	}

	if err := s.storage.Delete(ctx, p); err != nil {
		return fmt.Errorf("store: deleting layout %q: %w", id, err)
	}

	return nil
}

// Seed stores the layouts if no layout is stored yet, so the layouts of the user are never replaced.
func (s *LayoutStore) Seed(ctx context.Context, layouts []domain.Layout) error {
	existing, err := s.storage.List(ctx, layoutsFolder+"/")
	if err != nil {
		return fmt.Errorf("store: listing layouts: %w", err)
	}
	if len(existing) > 0 {
		return nil
	}

	for _, layout := range layouts {
		if err := s.Save(ctx, layout.ID, layout); err != nil {
			return err
		}
	}

	return nil
}
