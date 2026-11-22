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
	"time"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
)

const (
	// takenAtLayout formats the time a capture was taken at the end of its file name.
	takenAtLayout = "20060102-150405"
	// thumbnailImageIndex is the image index of the capture of a run that its thumbnail shows.
	thumbnailImageIndex = 1
)

// CaptureStore keeps the captures of the runs as JPEG files in the folder of their event,
// optionally their originals without the overlay, and a thumbnail of the first capture of a run.
type CaptureStore struct {
	storage media.Storage
	files   *RunFileIndex
}

// NewCaptureStore creates a CaptureStore.
func NewCaptureStore(storage media.Storage, files *RunFileIndex) *CaptureStore {
	return &CaptureStore{
		storage: storage,
		files:   files,
	}
}

// List returns every stored capture of every run, or their originals if originals is set.
func (s *CaptureStore) List(ctx context.Context, originals bool) ([]domain.CaptureFile, error) {
	return s.listFolder(ctx, captureFolder(originals))
}

// ListByRun returns the storage paths of the captures of a run,
// or of their originals if originals is set, ordered by image index.
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

// Find returns one stored capture of a run, or its original if original is set, or returns apperr.ErrNotFound.
func (s *CaptureStore) Find(ctx context.Context, run, imageIndex int, original bool) (domain.CaptureFile, error) {
	return s.findIn(ctx, captureFolder(original), run, imageIndex)
}

// Get reads one stored capture of a run, or its original if original is set, or returns apperr.ErrNotFound.
func (s *CaptureStore) Get(ctx context.Context, run, imageIndex int, original bool) ([]byte, error) {
	capture, err := s.Find(ctx, run, imageIndex, original)
	if err != nil {
		return nil, err
	}

	data, err := media.ReadAll(ctx, s.storage, capture.Path)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading %q: %w", capture.Path, err)
	}

	return data, nil
}

// FindThumbnail returns the thumbnail of a run, or else its first capture.
func (s *CaptureStore) FindThumbnail(ctx context.Context, run int) (domain.CaptureFile, error) {
	thumbnail, err := s.findIn(ctx, thumbnailsFolder, run, thumbnailImageIndex)
	if !errors.Is(err, apperr.ErrNotFound) {
		return thumbnail, err
	}

	return s.findIn(ctx, capturesFolder, run, thumbnailImageIndex)
}

// SaveThumbnail stores data as the thumbnail of the run of capture, which must be its first capture.
func (s *CaptureStore) SaveThumbnail(ctx context.Context, capture domain.Capture, data []byte) error {
	defer s.files.invalidate()

	thumbnailPath := path.Join(thumbnailsFolder, capture.Event, captureFileName(capture))
	if err := s.storage.Put(ctx, thumbnailPath, bytes.NewReader(data), int64(len(data))); err != nil {
		return fmt.Errorf("store: saving thumbnail of run %d: %w", capture.Run, err)
	}

	return nil
}

// Create stores data as the capture and, unless original is nil, original as its original.
// A capture whose original cannot be stored is removed again, so no capture lacks its original.
func (s *CaptureStore) Create(ctx context.Context, capture domain.Capture, data, original []byte) error {
	defer s.files.invalidate()

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
	return runFileName(capture.Prefix, capture.Run, capturePart(capture.ImageIndex), capture.TakenAt.Format(takenAtLayout))
}

// Delete removes the stored capture, original, and thumbnail of one image of a run, if there are any.
func (s *CaptureStore) Delete(ctx context.Context, run, imageIndex int) error {
	defer s.files.invalidate()

	for _, folder := range []string{capturesFolder, originalsFolder, thumbnailsFolder} {
		capture, err := s.findIn(ctx, folder, run, imageIndex)
		if errors.Is(err, apperr.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if err := s.storage.Delete(ctx, capture.Path); err != nil {
			return fmt.Errorf("store: deleting %q: %w", capture.Path, err)
		}
	}

	return nil
}

// findIn returns one capture file of a run below folder, or returns apperr.ErrNotFound.
func (s *CaptureStore) findIn(ctx context.Context, folder string, run, imageIndex int) (domain.CaptureFile, error) {
	captures, err := s.listFolder(ctx, folder)
	if err != nil {
		return domain.CaptureFile{}, err
	}

	for _, capture := range captures {
		if capture.Run == run && capture.ImageIndex == imageIndex {
			return capture, nil
		}
	}

	return domain.CaptureFile{}, apperr.ErrNotFound
}

// listFolder returns every capture file below folder.
func (s *CaptureStore) listFolder(ctx context.Context, folder string) ([]domain.CaptureFile, error) {
	files, err := s.files.list(ctx, folder)
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
			TakenAt:    parseTakenAt(file.suffix),
		})
	}

	return captures, nil
}

// parseTakenAt reads the time a capture was taken from the suffix of its file name in the local time zone,
// zero if the suffix is no such time.
func parseTakenAt(suffix string) time.Time {
	takenAt, err := time.ParseInLocation(takenAtLayout, suffix, time.Local)
	if err != nil {
		return time.Time{}
	}
	return takenAt
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
