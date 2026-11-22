package service

import (
	"context"
	"fmt"
	"io"
	"slices"
	"time"

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

// NewGalleryService creates a GalleryService. Every method waits for the capture captureService stores in the background,
// so it sees the latest capture and never deletes a run while it is written.
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
	// TakenAt is the time of the earliest capture of the run, zero if no file name holds one.
	TakenAt time.Time
}

// List returns one page of the stored runs that match the filter, the most recent first, and the number of all of them.
func (s *GalleryService) List(ctx context.Context, filter domain.RunListFilter) ([]RunSummary, int, error) {
	// The runs are ordered by the time they were taken, so they follow the days of ListDays even after the clock
	// of the booth went back. Runs of the same time follow their number, and runs without a time come last.
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
	slices.SortFunc(runs, compareRunsByTime)

	total := len(runs)
	start := min(filter.Offset, total)
	end := min(start+filter.Limit, total)

	return runs[start:end], total, nil
}

// compareRunsByTime orders runs by the time they were taken and then by number, the most recent first,
// with the runs without a time last.
func compareRunsByTime(a, b RunSummary) int {
	if a.TakenAt.IsZero() != b.TakenAt.IsZero() {
		if a.TakenAt.IsZero() {
			return 1
		}
		return -1
	}
	if order := b.TakenAt.Compare(a.TakenAt); order != 0 {
		return order
	}
	return b.Run - a.Run
}

// EventSummary describes the stored runs of one event folder.
type EventSummary struct {
	// Event is the slug of the event folder.
	Event string
	// Current is set for the event the booth files new captures under.
	Current       bool
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

// DaySummary counts the stored runs taken on one calendar day.
type DaySummary struct {
	// Day is the midnight in the local time zone that starts the day.
	Day      time.Time
	RunCount int
}

// ListDays counts the stored runs of an event, or of every event without one, per local calendar day of their TakenAt,
// the most recent day first. Runs without TakenAt are left out.
func (s *GalleryService) ListDays(ctx context.Context, event *string) ([]DaySummary, error) {
	runs, err := s.summarizeRuns(ctx)
	if err != nil {
		return nil, err
	}

	byDay := make(map[time.Time]int)
	for _, run := range runs {
		if run.TakenAt.IsZero() || (event != nil && run.Event != *event) {
			continue
		}
		local := run.TakenAt.In(time.Local)
		byDay[time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)]++
	}

	days := make([]DaySummary, 0, len(byDay))
	for day, count := range byDay {
		days = append(days, DaySummary{
			Day:      day,
			RunCount: count,
		})
	}
	slices.SortFunc(days, func(a, b DaySummary) int { return b.Day.Compare(a.Day) })

	return days, nil
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
		if !file.TakenAt.IsZero() && (run.TakenAt.IsZero() || file.TakenAt.Before(run.TakenAt)) {
			run.TakenAt = file.TakenAt
		}
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

// ListFiles returns every stored capture or original of the runs, ordered by run and image index,
// so a caller can answer unknown runs before streaming them with Zip.
func (s *GalleryService) ListFiles(ctx context.Context, runs []int, originals bool) ([]domain.CaptureFile, error) {
	return s.listFiles(ctx, originals, func(file domain.CaptureFile) bool { return slices.Contains(runs, file.Run) })
}

// ListEventFiles returns every stored capture or original of an event, ordered like ListFiles.
func (s *GalleryService) ListEventFiles(ctx context.Context, event string, originals bool) ([]domain.CaptureFile, error) {
	return s.listFiles(ctx, originals, func(file domain.CaptureFile) bool { return file.Event == event })
}

// listFiles returns the stored captures, or their originals, that match, ordered by run and image index,
// or returns apperr.ErrNotFound if none does.
func (s *GalleryService) listFiles(ctx context.Context, originals bool, match func(file domain.CaptureFile) bool) ([]domain.CaptureFile, error) {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return nil, err
	}

	all, err := s.captures.List(ctx, originals)
	if err != nil {
		return nil, fmt.Errorf("service: listing the files of the gallery: %w", err)
	}

	files := make([]domain.CaptureFile, 0)
	for _, file := range all {
		if match(file) {
			files = append(files, file)
		}
	}
	if len(files) == 0 {
		return nil, apperr.ErrNotFound
	}
	slices.SortFunc(files, func(a, b domain.CaptureFile) int {
		if a.Run != b.Run {
			return a.Run - b.Run
		}
		return a.ImageIndex - b.ImageIndex
	})

	return files, nil
}

// GetCapture returns one stored capture of a run, or its original without the overlay.
func (s *GalleryService) GetCapture(ctx context.Context, run, imageIndex int, original bool) (domain.CaptureFile, error) {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return domain.CaptureFile{}, err
	}

	return s.captures.Find(ctx, run, imageIndex, original)
}

// GetThumbnail returns the thumbnail of a run, or its first capture for a run stored before thumbnails existed.
// It returns apperr.ErrNotFound if the run does not exist.
func (s *GalleryService) GetThumbnail(ctx context.Context, run int) (domain.CaptureFile, error) {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return domain.CaptureFile{}, err
	}

	return s.captures.FindThumbnail(ctx, run)
}

// OpenFile opens a file returned by GetCapture or GetThumbnail for reading, or returns apperr.ErrNotFound
// if it was deleted since. The caller must close the returned reader.
func (s *GalleryService) OpenFile(ctx context.Context, file domain.CaptureFile) (io.ReadCloser, error) {
	return s.runs.OpenFile(ctx, file.Path)
}

// Delete removes every stored file of a run, or returns apperr.ErrNotFound if the run has no capture.
func (s *GalleryService) Delete(ctx context.Context, run int) error {
	if err := s.captureService.awaitStore(ctx); err != nil {
		return err
	}

	captures, err := s.captures.ListByRun(ctx, run, false)
	if err != nil {
		return fmt.Errorf("service: listing run %d: %w", run, err)
	}
	if len(captures) == 0 {
		return apperr.ErrNotFound
	}

	if err := s.runs.Delete(ctx, run); err != nil {
		return fmt.Errorf("service: deleting run %d: %w", run, err)
	}

	return nil
}

// Zip streams files, as listed by ListFiles or ListEventFiles, as a zip archive to w.
func (s *GalleryService) Zip(ctx context.Context, w io.Writer, files []domain.CaptureFile) error {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}

	return s.runs.ZipFiles(ctx, w, paths)
}
