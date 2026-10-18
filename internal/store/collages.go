package store

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
)

// CollageStore keeps the rendered collage pages of the runs as JPEG files in the folder of their event,
// named like the captures of the run.
type CollageStore struct {
	storage media.Storage
}

// NewCollageStore creates a CollageStore.
func NewCollageStore(storage media.Storage) *CollageStore {
	return &CollageStore{
		storage: storage,
	}
}

// Create stores data as the collage page.
func (s *CollageStore) Create(ctx context.Context, collage domain.Collage, data []byte) error {
	if err := s.storage.Put(ctx, s.Path(collage), bytes.NewReader(data), int64(len(data))); err != nil {
		return fmt.Errorf("store: storing collage for run %d page %d: %w", collage.Run, collage.Page, err)
	}

	return nil
}

// Delete removes every collage page of a run.
func (s *CollageStore) Delete(ctx context.Context, run int) error {
	if err := deleteRunFiles(ctx, s.storage, collagesFolder, run); err != nil {
		return fmt.Errorf("store: removing the collages of run %d: %w", run, err)
	}

	return nil
}

// Exists reports whether every collage page is stored.
func (s *CollageStore) Exists(ctx context.Context, collages []domain.Collage) (bool, error) {
	for _, collage := range collages {
		exists, err := s.storage.Exists(ctx, s.Path(collage))
		if err != nil {
			return false, fmt.Errorf("store: checking collage for run %d page %d: %w", collage.Run, collage.Page, err)
		}
		if !exists {
			return false, nil
		}
	}

	return true, nil
}

// Path returns the storage path of a collage page, next to the other collages of the event.
func (s *CollageStore) Path(collage domain.Collage) string {
	part := fmt.Sprintf("collage-%d", collage.Page+1)
	return path.Join(collagesFolder, collage.Event, runFileName(collage.Prefix, collage.Run, part, collage.LayoutHash))
}
