package store

import (
	"context"
	"slices"
	"strings"

	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
)

// layoutsFolder is the storage folder the layouts are kept in.
const layoutsFolder = "layouts"

// LayoutStore keeps every print layout as a JSON file in the storage, named after the layout.
type LayoutStore struct {
	files *jsonFiles[domain.Layout]
}

// NewLayoutStore creates a LayoutStore.
func NewLayoutStore(storage media.Storage) *LayoutStore {
	return &LayoutStore{
		files: newJSONFiles[domain.Layout](storage, layoutsFolder, "layout"),
	}
}

// List returns all layouts, sorted by name, and fails if one of them cannot be read.
func (s *LayoutStore) List(ctx context.Context) ([]domain.Layout, error) {
	layouts, err := s.files.list(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(layouts, func(a, b domain.Layout) int { return strings.Compare(a.Name, b.Name) })

	return layouts, nil
}

// Get returns the layout with the given name, or apperr.ErrNotFound if it does not exist.
// A name that cannot address a layout file fails with ErrInvalidName.
func (s *LayoutStore) Get(ctx context.Context, name string) (*domain.Layout, error) {
	return s.files.get(ctx, name)
}

// Save writes the layout under the given name, replacing an existing one.
// A name that cannot address a layout file fails with ErrInvalidName.
func (s *LayoutStore) Save(ctx context.Context, name string, layout domain.Layout) error {
	return s.files.save(ctx, name, layout)
}

// Delete removes the layout with the given name, or returns apperr.ErrNotFound if it does not exist.
// A name that cannot address a layout file fails with ErrInvalidName.
func (s *LayoutStore) Delete(ctx context.Context, name string) error {
	return s.files.delete(ctx, name)
}

// Seed stores the layouts if no layout is stored yet, so the layouts of the user are never replaced.
func (s *LayoutStore) Seed(ctx context.Context, layouts []domain.Layout) error {
	existing, err := s.files.names(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	for _, layout := range layouts {
		if err := s.Save(ctx, layout.Name, layout); err != nil {
			return err
		}
	}

	return nil
}
