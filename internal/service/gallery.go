package service

import (
	"context"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/store"
)

// GalleryService lists, deletes, and archives the stored runs.
type GalleryService struct {
	runs           *store.RunStore
	captures       *store.CaptureStore
	captureService *CaptureService
}

// NewGalleryService creates a GalleryService. Every method waits for the capture captureService stores
// in the background, so it sees the latest capture and never deletes a run while it is written.
func NewGalleryService(runs *store.RunStore, captures *store.CaptureStore, captureService *CaptureService) *GalleryService {
	return &GalleryService{
		runs:           runs,
		captures:       captures,
		captureService: captureService,
	}
}

// RunSummary describes one run for a paginated gallery listing.
type RunSummary struct {
	Run int
	// Event is the slug of the event folder the run lies in.
	Event      string
	ImageCount int
	// OriginalCount is the number of captures stored without the overlay as well.
	OriginalCount int
}

// List returns one page of the stored runs that match the filter, the most recent first, and the number of all of them.
func (s *GalleryService) List(ctx context.Context, filter domain.RunListFilter) ([]RunSummary, int, error) {
	all, err := s.summarizeRuns(ctx)
	if err != nil {
		return nil, 0, err
	}

	runs := make([]RunSummary, 0, len(all))
	for _, run := range all {
		if filter.Event == nil || run.Event == *filter.Event {
			runs = append(runs, run)
		}
	}
	slices.SortFunc(runs, func(a, b RunSummary) int { return b.Run - a.Run })

	total := len(runs)
	start := min(filter.Offset, total)
	end := min(start+filter.Limit, total)

	return runs[start:end], total, nil
}

// EventSummary describes the stored runs of one event folder.
type EventSummary struct {
	// Event is the slug of the event folder.
	Event string
	// Current is set for the event the booth files new captures under.
	Current       bool
	RunCount      int
	ImageCount    int
	OriginalCount int
	// LastRun is the highest run number of the event, which orders the events by time.
	LastRun int
}

// ListEvents returns every event with stored runs, the most recent first.
func (s *GalleryService) ListEvents(ctx context.Context) ([]EventSummary, error) {
	runs, err := s.summarizeRuns(ctx)
	if err != nil {
		return nil, err
	}

	byEvent := make(map[string]*EventSummary)
	for _, run := range runs {
		event, ok := byEvent[run.Event]
		if !ok {
			event = &EventSummary{
				Event:   run.Event,
				Current: run.Event == s.captureService.eventSlug,
			}
			byEvent[run.Event] = event
		}
		event.RunCount++
		event.ImageCount += run.ImageCount
		event.OriginalCount += run.OriginalCount
		event.LastRun = max(event.LastRun, run.Run)
	}

	events := make([]EventSummary, 0, len(byEvent))
	for _, event := range byEvent {
		events = append(events, *event)
	}
	slices.SortFunc(events, func(a, b EventSummary) int { return b.LastRun - a.LastRun })

	return events, nil
}

// summarizeRuns counts the captures and originals of every stored run,
// after the capture being stored in the background is written.
func (s *GalleryService) summarizeRuns(ctx context.Context) ([]RunSummary, error) {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return nil, err
	}

	captures, err := s.captures.List(ctx, false)
	if err != nil {
		return nil, err
	}
	originals, err := s.captures.List(ctx, true)
	if err != nil {
		return nil, err
	}

	byRun := make(map[int]*RunSummary)
	for _, file := range captures {
		run, ok := byRun[file.Run]
		if !ok {
			run = &RunSummary{
				Run:   file.Run,
				Event: file.Event,
			}
			byRun[file.Run] = run
		}
		run.ImageCount++
	}
	for _, file := range originals {
		if run, ok := byRun[file.Run]; ok {
			run.OriginalCount++
		}
	}

	runs := make([]RunSummary, 0, len(byRun))
	for _, run := range byRun {
		runs = append(runs, *run)
	}
	return runs, nil
}

// ListFiles returns the storage paths of every stored capture of a run, or of its originals if originals is set,
// so a caller can answer an unknown run before it starts streaming them with Zip.
// It returns apperr.ErrNotFound if the run has no such file.
func (s *GalleryService) ListFiles(ctx context.Context, run int, originals bool) ([]string, error) {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return nil, err
	}

	paths, err := s.captures.ListByRun(ctx, run, originals)
	if err != nil {
		return nil, fmt.Errorf("service: listing run %d files: %w", run, err)
	}
	if len(paths) == 0 {
		return nil, apperr.ErrNotFound
	}

	return paths, nil
}

// GetCapture reads one stored capture of a run, or its original without the overlay if original is set.
// It returns apperr.ErrNotFound if it does not exist.
func (s *GalleryService) GetCapture(ctx context.Context, run, imageIndex int, original bool) ([]byte, error) {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return nil, err
	}

	return s.captures.Get(ctx, run, imageIndex, original)
}

// Delete removes every stored file of a run.
func (s *GalleryService) Delete(ctx context.Context, run int) error {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return err
	}

	if err := s.runs.Delete(ctx, run); err != nil {
		return fmt.Errorf("service: deleting run %d: %w", run, err)
	}

	return nil
}

// Zip streams the files of a run at paths, as listed by ListFiles, as a zip archive to w.
func (s *GalleryService) Zip(ctx context.Context, w io.Writer, paths []string) error {
	return s.runs.ZipFiles(ctx, w, paths)
}

// ZipEvent streams every stored capture of an event, or their originals if originals is set,
// as one zip archive to w, ordered by run and image index.
func (s *GalleryService) ZipEvent(ctx context.Context, w io.Writer, event string, originals bool) error {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return err
	}

	files, err := s.captures.List(ctx, originals)
	if err != nil {
		return fmt.Errorf("service: listing the files of event %q: %w", event, err)
	}

	paths := make([]string, 0, len(files))
	for _, file := range files {
		if file.Event == event {
			paths = append(paths, file.Path)
		}
	}
	// The file names start with the run number and the image index, so they sort by both
	slices.SortFunc(paths, func(a, b string) int { return strings.Compare(path.Base(a), path.Base(b)) })

	return s.runs.ZipFiles(ctx, w, paths)
}
