package service

import (
	"context"
	"embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/store"
)

//go:embed layouts/defaults/*.json
var defaultLayoutsFS embed.FS

// LayoutService manages the print layouts.
type LayoutService struct {
	store *store.LayoutStore
}

// NewLayoutService creates a LayoutService.
func NewLayoutService(store *store.LayoutStore) *LayoutService {
	return &LayoutService{
		store: store,
	}
}

// List returns all layouts, sorted by ID.
func (s *LayoutService) List(ctx context.Context) ([]domain.Layout, error) {
	return s.store.List(ctx)
}

// Get returns the layout with the given ID, or apperr.ErrNotFound if it does not exist
// or id cannot address a layout file.
func (s *LayoutService) Get(ctx context.Context, id string) (*domain.Layout, error) {
	layout, err := s.store.Get(ctx, id)
	if errors.Is(err, store.ErrInvalidName) {
		return nil, apperr.ErrNotFound
	}
	return layout, err
}

// Save creates or replaces the layout with the given ID, which has to match the layout's own ID,
// so a stored layout always carries its file name.
func (s *LayoutService) Save(ctx context.Context, id string, layout domain.Layout) error {
	if layout.ID != id {
		return apperr.NewValidationError("id", "invalid")
	}

	err := s.store.Save(ctx, id, layout)
	if errors.Is(err, store.ErrInvalidName) {
		return apperr.NewValidationError("id", "invalid")
	}
	return err
}

// Delete removes the layout with the given ID, or returns apperr.ErrNotFound if it does not exist
// or id cannot address a layout file.
func (s *LayoutService) Delete(ctx context.Context, id string) error {
	err := s.store.Delete(ctx, id)
	if errors.Is(err, store.ErrInvalidName) {
		return apperr.ErrNotFound
	}
	return err
}

// EnsureDefaults stores the built-in layouts with their names translated into locale if no layout is stored yet.
func (s *LayoutService) EnsureDefaults(ctx context.Context, translator *i18n.Bundle, locale string) error {
	entries, err := defaultLayoutsFS.ReadDir("layouts/defaults")
	if err != nil {
		return fmt.Errorf("service: reading embedded default layouts: %w", err)
	}

	layouts := make([]domain.Layout, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := defaultLayoutsFS.ReadFile(path.Join("layouts/defaults", entry.Name()))
		if err != nil {
			return fmt.Errorf("service: reading embedded default layout %q: %w", entry.Name(), err)
		}

		var layout domain.Layout
		if err := json.Unmarshal(data, &layout); err != nil {
			return fmt.Errorf("service: parsing embedded default layout %q: %w", entry.Name(), err)
		}

		// T returns the key itself when no translation exists, the embedded English name then stays
		key := "layout." + layout.ID
		if translated := translator.T(locale, key, nil); translated != key {
			layout.Name = translated
		}
		layouts = append(layouts, layout)
	}

	if err := s.store.Seed(ctx, layouts); err != nil {
		return fmt.Errorf("service: seeding default layouts: %w", err)
	}

	return nil
}
