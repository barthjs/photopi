package store

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/media"
)

// Every file of a run lives flat in the folder of its event. The run number and the part identify the file,
// the timestamp only informs, since the clock of a booth without internet may be wrong.
// Run numbers come from a counter that never goes back, so a deleted run never passes its number on.

const (
	// capturesFolder is the storage folder the captures are kept in.
	capturesFolder = "captures"
	// originalsFolder holds the unmodified copies of the captures with the same folders and file names as capturesFolder.
	// It is a separate top-level folder, so the listings of the captures never include the originals.
	originalsFolder = "originals"
	// collagesFolder is the storage folder the rendered collage pages are kept in.
	collagesFolder = "prints"
	// thumbnailsFolder holds the thumbnails of the captures with the same folders and file names as capturesFolder.
	thumbnailsFolder = "thumbnails"

	// runCounterPath stores the last run number handed out. List skips files starting with a dot,
	// so the counter never shows up as a capture.
	runCounterPath = capturesFolder + "/.run-counter"
)

// runFilePattern matches the name of a run file with its run number, part, and suffix.
var runFilePattern = regexp.MustCompile(`_(\d{6})-([0-9a-z-]+)_([0-9a-z-]+)\.jpg$`)

// RunStore keeps the run counter and serves the files of the runs, its captures and its collage pages.
type RunStore struct {
	storage media.Storage
	files   *RunFileIndex
}

// NewRunStore creates a RunStore.
func NewRunStore(storage media.Storage, files *RunFileIndex) *RunStore {
	return &RunStore{
		storage: storage,
		files:   files,
	}
}

// GetFile reads the file of a run at a storage path, or returns apperr.ErrNotFound.
func (s *RunStore) GetFile(ctx context.Context, p string) ([]byte, error) {
	data, err := media.ReadAll(ctx, s.storage, p)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading %q: %w", p, err)
	}

	return data, nil
}

// OpenFile opens the file of a run at a storage path for reading, or returns apperr.ErrNotFound.
// The caller must close the returned reader.
func (s *RunStore) OpenFile(ctx context.Context, p string) (io.ReadCloser, error) {
	reader, err := s.storage.Get(ctx, p)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading %q: %w", p, err)
	}

	return reader, nil
}

// Create hands out the next run number, which callers serialize, and stores it in the counter.
func (s *RunStore) Create(ctx context.Context) (int, error) {
	// Without a counter, it continues after the highest stored run, so a number is never handed out twice
	last, err := s.readCounter(ctx)
	if err != nil {
		return 0, err
	}

	files, err := s.files.list(ctx, capturesFolder)
	if err != nil {
		return 0, err
	}
	for _, file := range files {
		last = max(last, file.run)
	}

	next := last + 1
	data := []byte(strconv.Itoa(next))
	if err := s.storage.Put(ctx, runCounterPath, bytes.NewReader(data), int64(len(data))); err != nil {
		return 0, fmt.Errorf("store: storing run counter: %w", err)
	}

	return next, nil
}

// readCounter returns the last run number handed out, or 0 if no counter is stored yet.
func (s *RunStore) readCounter(ctx context.Context) (int, error) {
	data, err := media.ReadAll(ctx, s.storage, runCounterPath)
	if errors.Is(err, media.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("store: reading run counter: %w", err)
	}

	last, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// A damaged counter is replaced, the scan of the stored runs keeps the numbers unique
		return 0, nil //nolint:nilerr // A damaged counter counts as no counter.
	}

	return last, nil
}

// Delete removes every file of a run, its captures, originals, thumbnails, and collage pages.
func (s *RunStore) Delete(ctx context.Context, run int) error {
	defer s.files.invalidate()

	for _, folder := range []string{capturesFolder, originalsFolder, thumbnailsFolder} {
		files, err := s.files.list(ctx, folder)
		if err != nil {
			return err
		}
		if err := deleteRunFiles(ctx, s.storage, files, run); err != nil {
			return err
		}
	}

	// The collage pages are written by the CollageStore, which keeps no index
	files, err := listRunFiles(ctx, s.storage, collagesFolder)
	if err != nil {
		return err
	}
	return deleteRunFiles(ctx, s.storage, files, run)
}

// SignFile returns a URL that serves the file of a run at a storage path for ttl.
func (s *RunStore) SignFile(ctx context.Context, p string, ttl time.Duration) (string, error) {
	url, err := s.storage.SignedURL(ctx, p, ttl)
	if err != nil {
		return "", fmt.Errorf("store: signing %q: %w", p, err)
	}

	return url, nil
}

// ZipFiles streams the files of runs at the storage paths as one uncompressed zip archive to w.
func (s *RunStore) ZipFiles(ctx context.Context, w io.Writer, paths []string) error {
	zw := zip.NewWriter(w)
	defer func(zw *zip.Writer) { _ = zw.Close() }(zw)
	for _, p := range paths {
		if err := s.addZipEntry(ctx, zw, p); err != nil {
			return err
		}
	}

	return zw.Close()
}

// addZipEntry copies the file at p into a new, uncompressed entry of zw.
func (s *RunStore) addZipEntry(ctx context.Context, zw *zip.Writer, p string) error {
	reader, err := s.storage.Get(ctx, p)
	if err != nil {
		return fmt.Errorf("store: reading %q: %w", p, err)
	}
	defer func() { _ = reader.Close() }()

	entry, err := zw.CreateHeader(&zip.FileHeader{
		Name: path.Base(p),
		// JPEGs do not shrink any further
		Method: zip.Store,
	})
	if err != nil {
		return fmt.Errorf("store: adding %q to zip: %w", p, err)
	}
	if _, err := io.Copy(entry, reader); err != nil {
		return fmt.Errorf("store: writing %q to zip: %w", p, err)
	}

	return nil
}

// runFile is a file of a run with the parts of its path and name.
type runFile struct {
	path string
	// event is the slug of the event folder the file lies in.
	event string
	run   int
	part  string
	// suffix ends the file name, the time a capture was taken or the hash of a collage page.
	suffix string
}

// runFileName joins the prefix, the run number, the part, and the suffix into a file name.
// A separator the prefix already ends with is dropped, so "Party_" and "Party" give the same name.
func runFileName(prefix string, run int, part string, suffix string) string {
	return fmt.Sprintf("%s_%06d-%s_%s.jpg", strings.TrimRight(prefix, "_- "), run, part, suffix)
}

// deleteRunFiles removes the files of a run among files.
func deleteRunFiles(ctx context.Context, storage media.Storage, files []runFile, run int) error {
	for _, file := range files {
		if file.run != run {
			continue
		}
		if err := storage.Delete(ctx, file.path); err != nil {
			return fmt.Errorf("store: deleting %q: %w", file.path, err)
		}
	}

	return nil
}

// listRunFiles returns every run file below folder/<event-slug>/, across all event folders,
// since the event name may have changed since a run was taken. Files nested deeper are ignored.
func listRunFiles(ctx context.Context, storage media.Storage, folder string) ([]runFile, error) {
	paths, err := storage.List(ctx, folder+"/")
	if err != nil {
		return nil, err
	}

	files := make([]runFile, 0, len(paths))
	for _, p := range paths {
		rel := strings.TrimPrefix(p, folder+"/")
		if strings.Count(rel, "/") != 1 {
			continue
		}

		run, part, suffix, ok := parseRunFile(path.Base(rel))
		if !ok {
			continue
		}

		files = append(files, runFile{
			path:   p,
			event:  path.Dir(rel),
			run:    run,
			part:   part,
			suffix: suffix,
		})
	}

	return files, nil
}

// parseRunFile returns the run, the part, and the suffix of a file name, ok is false if it is no run file.
func parseRunFile(name string) (run int, part string, suffix string, ok bool) {
	match := runFilePattern.FindStringSubmatch(name)
	if match == nil {
		return 0, "", "", false
	}

	run, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, "", "", false
	}

	return run, match[2], match[3], true
}
