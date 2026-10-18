package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
)

// CaptureStore keeps the captures of the runs as JPEG files in the folder of their event,
// and optionally their originals without the overlay.
type CaptureStore struct {
	storage media.Storage
}

// NewCaptureStore creates a CaptureStore.
func NewCaptureStore(storage media.Storage) *CaptureStore {
	return &CaptureStore{
		storage: storage,
	}
}

// List returns every stored capture of every run, or their originals if originals is set.
func (s *CaptureStore) List(ctx context.Context, originals bool) ([]domain.CaptureFile, error) {
	files, err := listRunFiles(ctx, s.storage, captureFolder(originals))
	if err != nil {
		return nil, fmt.Errorf("store: listing captures: %w", err)
	}

	captures := make([]domain.CaptureFile, 0, len(files))
	for _, file := range files {
		index, err := strconv.Atoi(file.part)
		if err != nil {
			continue
		}
		captures = append(captures, domain.CaptureFile{
			Path:       file.path,
			Event:      file.event,
			Run:        file.run,
			ImageIndex: index,
		})
	}

	return captures, nil
}

// ListByRun returns the storage paths of the captures of a run, or of their originals if originals is set,
// ordered by image index.
func (s *CaptureStore) ListByRun(ctx context.Context, run int, originals bool) ([]string, error) {
	all, err := s.List(ctx, originals)
	if err != nil {
		return nil, err
	}

	captures := make([]domain.CaptureFile, 0)
	for _, capture := range all {
		if capture.Run == run {
			captures = append(captures, capture)
		}
	}
	slices.SortFunc(captures, func(a, b domain.CaptureFile) int {
		if a.ImageIndex != b.ImageIndex {
			return a.ImageIndex - b.ImageIndex
		}
		return strings.Compare(a.Path, b.Path)
	})

	paths := make([]string, 0, len(captures))
	for _, capture := range captures {
		paths = append(paths, capture.Path)
	}

	return paths, nil
}

// Get reads one stored capture of a run, or its original if original is set, or returns apperr.ErrNotFound.
func (s *CaptureStore) Get(ctx context.Context, run, imageIndex int, original bool) ([]byte, error) {
	capturePath, found, err := s.find(ctx, captureFolder(original), run, imageIndex)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, apperr.ErrNotFound
	}

	data, err := media.ReadAll(ctx, s.storage, capturePath)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading %q: %w", capturePath, err)
	}

	return data, nil
}

// Create stores data as the capture and, unless original is nil, original as its original.
// A capture whose original cannot be stored is removed again, so no capture lacks its original.
func (s *CaptureStore) Create(ctx context.Context, capture domain.Capture, data, original []byte) error {
	fileName := captureFileName(capture)

	capturePath := path.Join(capturesFolder, capture.Event, fileName)
	if err := s.storage.Put(ctx, capturePath, bytes.NewReader(data), int64(len(data))); err != nil {
		return fmt.Errorf("store: saving run %d image %d: %w", capture.Run, capture.ImageIndex, err)
	}

	if original == nil {
		return nil
	}

	originalPath := path.Join(originalsFolder, capture.Event, fileName)
	if err := s.storage.Put(ctx, originalPath, bytes.NewReader(original), int64(len(original))); err != nil {
		_ = s.storage.Delete(ctx, capturePath)
		return fmt.Errorf("store: saving original for run %d image %d: %w", capture.Run, capture.ImageIndex, err)
	}

	return nil
}

// captureFileName returns the file name of a capture.
func captureFileName(capture domain.Capture) string {
	return runFileName(capture.Prefix, capture.Run, capturePart(capture.ImageIndex), capture.TakenAt.Format("20060102-150405"))
}

// Delete removes the stored capture and original of one image of a run, if there are any.
func (s *CaptureStore) Delete(ctx context.Context, run, imageIndex int) error {
	for _, folder := range []string{capturesFolder, originalsFolder} {
		capturePath, found, err := s.find(ctx, folder, run, imageIndex)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if err := s.storage.Delete(ctx, capturePath); err != nil {
			return fmt.Errorf("store: deleting %q: %w", capturePath, err)
		}
	}

	return nil
}

// find returns the path of one capture of a run below folder, found is false if it does not exist.
func (s *CaptureStore) find(ctx context.Context, folder string, run, imageIndex int) (capturePath string, found bool, err error) {
	files, err := listRunFiles(ctx, s.storage, folder)
	if err != nil {
		return "", false, fmt.Errorf("store: listing captures: %w", err)
	}

	part := capturePart(imageIndex)
	for _, file := range files {
		if file.run == run && file.part == part {
			return file.path, true, nil
		}
	}

	return "", false, nil
}

// capturePart returns the part of a capture file name for an image index.
func capturePart(index int) string {
	return fmt.Sprintf("%02d", index)
}

// captureFolder returns the storage folder of the captures, or of their originals if originals is set.
func captureFolder(originals bool) string {
	if originals {
		return originalsFolder
	}
	return capturesFolder
}
