package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
)

func TestGalleryService_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the runs", func(t *testing.T) {
		t.Parallel()

		svc := newTestGalleryService(putGalleryRuns(t))

		runs, total, err := svc.List(t.Context(), domain.RunListFilter{
			Limit:  10,
			Offset: 0,
		})

		require.NoError(t, err)
		assert.Equal(t, []RunSummary{
			{
				Run:        5,
				Event:      "event-1",
				ImageCount: 5,
				TakenAt:    galleryRunsTakenAt,
			},
			{
				Run:        3,
				Event:      "event-1",
				ImageCount: 3,
				TakenAt:    galleryRunsTakenAt,
			},
			{
				Run:        2,
				Event:      "event-0",
				ImageCount: 2,
				TakenAt:    galleryRunsTakenAt,
			},
			{
				Run:        1,
				Event:      "event-1",
				ImageCount: 1,
				TakenAt:    galleryRunsTakenAt,
			},
		}, runs)
		assert.Equal(t, 4, total)
	})

	t.Run("filters by event", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/old-party/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/summer-party/x_000002-01_20260101-000000.jpg", []byte("x"))

		runs, total, err := newTestGalleryService(storage).List(t.Context(), domain.RunListFilter{
			Event:  new("old-party"),
			Limit:  10,
			Offset: 0,
		})

		require.NoError(t, err)
		require.Len(t, runs, 1)
		assert.Equal(t, 1, runs[0].Run)
		assert.Equal(t, 1, total)
	})

	t.Run("paginates", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			offset   int
			wantRuns []RunSummary
		}{
			{
				name:   "returns the most recent runs from the start",
				offset: 0,
				wantRuns: []RunSummary{
					{
						Run:        5,
						Event:      "event-1",
						ImageCount: 5,
						TakenAt:    galleryRunsTakenAt,
					},
					{
						Run:        3,
						Event:      "event-1",
						ImageCount: 3,
						TakenAt:    galleryRunsTakenAt,
					},
				},
			},
			{
				name:   "skips the aborted run at the end",
				offset: 2,
				wantRuns: []RunSummary{
					{
						Run:        2,
						Event:      "event-0",
						ImageCount: 2,
						TakenAt:    galleryRunsTakenAt,
					},
					{
						Run:        1,
						Event:      "event-1",
						ImageCount: 1,
						TakenAt:    galleryRunsTakenAt,
					},
				},
			},
			{
				name:     "returns no runs beyond the last run",
				offset:   4,
				wantRuns: []RunSummary{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				svc := newTestGalleryService(putGalleryRuns(t))

				runs, total, err := svc.List(t.Context(), domain.RunListFilter{
					Limit:  2,
					Offset: tt.offset,
				})

				require.NoError(t, err)
				assert.Equal(t, tt.wantRuns, runs)
				assert.Equal(t, 4, total)
			})
		}
	})

	t.Run("takes the time of the earliest capture", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_20260101-120000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000001-02_20260101-110000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000001-03_unknown.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000002-01_unknown.jpg", []byte("x"))

		runs, _, err := newTestGalleryService(storage).List(t.Context(), domain.RunListFilter{
			Limit:  10,
			Offset: 0,
		})

		require.NoError(t, err)
		require.Len(t, runs, 2)
		assert.Equal(t, time.Date(2026, 1, 1, 11, 0, 0, 0, time.Local), runs[0].TakenAt)
		assert.True(t, runs[1].TakenAt.IsZero(), "a run without a time in its file names has none")
	})

	t.Run("orders the runs by the time they were taken", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		// The clock of the booth went back between runs 1 and 2
		putFile(t, storage, "captures/a/x_000001-01_20260102-120000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000002-01_20260101-120000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000003-01_20260101-120000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000004-01_unknown.jpg", []byte("x"))

		runs, _, err := newTestGalleryService(storage).List(t.Context(), domain.RunListFilter{
			Limit:  10,
			Offset: 0,
		})

		require.NoError(t, err)
		order := make([]int, 0, len(runs))
		for _, run := range runs {
			order = append(order, run.Run)
		}
		assert.Equal(t, []int{1, 3, 2, 4}, order, "runs of the same time follow their number and runs without a time come last")
	})

	t.Run("returns no runs for an empty gallery", func(t *testing.T) {
		t.Parallel()

		svc := newTestGalleryService(newTestStorage(t))

		runs, total, err := svc.List(t.Context(), domain.RunListFilter{
			Limit:  10,
			Offset: 0,
		})

		require.NoError(t, err)
		assert.Equal(t, []RunSummary{}, runs)
		assert.Equal(t, 0, total)
	})
}

// galleryRunsTakenAt is the time in the file names of putGalleryRuns.
var galleryRunsTakenAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)

// putGalleryRuns returns a storage with runs 1, 2, 3, and 5 across two events,
// where run 4 was aborted and left nothing behind.
func putGalleryRuns(t *testing.T) media.Storage {
	t.Helper()

	storage := newTestStorage(t)
	for _, run := range []int{1, 2, 3, 5} {
		for index := 1; index <= run; index++ {
			name := fmt.Sprintf("captures/event-%d/x_%06d-%02d_20260101-000000.jpg", run%2, run, index)
			putFile(t, storage, name, []byte("x"))
		}
	}
	return storage
}

func TestGalleryService_ListEvents(t *testing.T) {
	t.Parallel()

	t.Run("returns the events", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/old-party/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/old-party/x_000001-02_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/summer-party/x_000002-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/summer-party/x_000002-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "prints/summer-party/x_000002-collage-1_0a1b2c3d.jpg", []byte("x"))

		events, err := newTestGalleryService(storage).ListEvents(t.Context())

		require.NoError(t, err)
		assert.Equal(t, []EventSummary{
			{
				Event:         "summer-party",
				Current:       true,
				ImageCount:    1,
				OriginalCount: 1,
				LastRun:       2,
			},
			{
				Event:      "old-party",
				ImageCount: 2,
				LastRun:    1,
			},
		}, events, "the most recent event comes first, and the event of the booth is marked")
	})
}

func TestGalleryService_ListDays(t *testing.T) {
	t.Parallel()

	t.Run("counts the runs per day", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_20260101-090000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000002-01_20260102-235959.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000003-01_20260102-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000003-02_20260103-000000.jpg", []byte("x"))

		days, err := newTestGalleryService(storage).ListDays(t.Context(), nil)

		require.NoError(t, err)
		assert.Equal(t, []DaySummary{
			{
				Day:      time.Date(2026, 1, 2, 0, 0, 0, 0, time.Local),
				RunCount: 2,
			},
			{
				Day:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
				RunCount: 1,
			},
		}, days, "the most recent day comes first, and a run counts on the day of its earliest capture")
	})

	t.Run("filters by event", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/old-party/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/summer-party/x_000002-01_20260102-000000.jpg", []byte("x"))

		days, err := newTestGalleryService(storage).ListDays(t.Context(), new("old-party"))

		require.NoError(t, err)
		assert.Equal(t, []DaySummary{
			{
				Day:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
				RunCount: 1,
			},
		}, days)
	})

	t.Run("leaves out runs without a time", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_unknown.jpg", []byte("x"))

		days, err := newTestGalleryService(storage).ListDays(t.Context(), nil)

		require.NoError(t, err)
		assert.Empty(t, days)
	})
}

func TestGalleryService_ListEventFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures of the event ordered by run", func(t *testing.T) {
		t.Parallel()

		svc := newTestGalleryService(putGalleryEvents(t))

		files, err := svc.ListEventFiles(t.Context(), "party", false)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"captures/party/x_000001-01_20260101-000000.jpg",
			"captures/party/x_000002-01_20260101-000000.jpg",
		}, capturePaths(files))
	})

	t.Run("returns the originals", func(t *testing.T) {
		t.Parallel()

		svc := newTestGalleryService(putGalleryEvents(t))

		files, err := svc.ListEventFiles(t.Context(), "party", true)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"originals/party/x_000001-01_20260101-000000.jpg",
		}, capturePaths(files))
	})

	t.Run("returns not found for an unknown event", func(t *testing.T) {
		t.Parallel()

		svc := newTestGalleryService(putGalleryEvents(t))

		_, err := svc.ListEventFiles(t.Context(), "unknown", false)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestGalleryService_ListFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures of the runs ordered by run and image", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/b/x_000005-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/y_000003-02_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/y_000003-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000004-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		files, err := svc.ListFiles(t.Context(), []int{5, 3, 9}, false)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"captures/a/y_000003-01_20260101-000000.jpg",
			"captures/a/y_000003-02_20260101-000000.jpg",
			"captures/b/x_000005-01_20260101-000000.jpg",
		}, capturePaths(files), "unknown runs are left out and the prefix of the file names does not change the order")
	})

	t.Run("returns the originals", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("composited"))
		putFile(t, storage, "originals/a/x_000003-01_20260101-000000.jpg", []byte("original"))
		svc := newTestGalleryService(storage)

		files, err := svc.ListFiles(t.Context(), []int{3}, true)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"originals/a/x_000003-01_20260101-000000.jpg",
		}, capturePaths(files))
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		_, err := svc.ListFiles(t.Context(), []int{4}, false)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for a run without originals", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		_, err := svc.ListFiles(t.Context(), []int{3}, true)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestGalleryService_GetCapture(t *testing.T) {
	t.Parallel()

	t.Run("returns the capture", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("composited"))
		putFile(t, storage, "originals/a/x_000003-01_20260101-000000.jpg", []byte("original"))
		svc := newTestGalleryService(storage)

		file, err := svc.GetCapture(t.Context(), 3, 1, false)

		require.NoError(t, err)
		assert.Equal(t, "captures/a/x_000003-01_20260101-000000.jpg", file.Path)
	})

	t.Run("returns the original", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("composited"))
		putFile(t, storage, "originals/a/x_000003-01_20260101-000000.jpg", []byte("original"))
		svc := newTestGalleryService(storage)

		file, err := svc.GetCapture(t.Context(), 3, 1, true)

		require.NoError(t, err)
		assert.Equal(t, "originals/a/x_000003-01_20260101-000000.jpg", file.Path)
	})

	t.Run("returns not found for an unknown capture", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		_, err := svc.GetCapture(t.Context(), 3, 2, false)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestGalleryService_GetThumbnail(t *testing.T) {
	t.Parallel()

	t.Run("returns the thumbnail of the run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "thumbnails/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		file, err := svc.GetThumbnail(t.Context(), 3)

		require.NoError(t, err)
		assert.Equal(t, "thumbnails/a/x_000003-01_20260101-000000.jpg", file.Path)
	})

	t.Run("returns the first capture of a run without a thumbnail", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		file, err := svc.GetThumbnail(t.Context(), 3)

		require.NoError(t, err)
		assert.Equal(t, "captures/a/x_000003-01_20260101-000000.jpg", file.Path)
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		svc := newTestGalleryService(newTestStorage(t))

		_, err := svc.GetThumbnail(t.Context(), 3)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestGalleryService_OpenFile(t *testing.T) {
	t.Parallel()

	t.Run("returns the content of the file", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", content)
		svc := newTestGalleryService(storage)
		file, err := svc.GetCapture(t.Context(), 3, 1, false)
		require.NoError(t, err)

		reader, err := svc.OpenFile(t.Context(), file)

		require.NoError(t, err)
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		assert.Equal(t, content, data)
	})

	t.Run("returns not found for a file deleted since", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)
		file, err := svc.GetCapture(t.Context(), 3, 1, false)
		require.NoError(t, err)
		require.NoError(t, storage.Delete(t.Context(), file.Path))

		_, err = svc.OpenFile(t.Context(), file)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestGalleryService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the run from the storage and the gallery", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000001-02_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "thumbnails/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "prints/a/x_000001-collage-1_0a1b2c3d.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)
		_, _, err := svc.List(t.Context(), domain.RunListFilter{
			Limit: 10,
		})
		require.NoError(t, err)

		err = svc.Delete(t.Context(), 1)

		require.NoError(t, err)
		assert.Equal(t, []string{"captures/a/x_000002-01_20260101-000000.jpg"}, listFiles(t, storage, "captures/"))
		assert.Empty(t, listFiles(t, storage, "originals/"))
		assert.Empty(t, listFiles(t, storage, "thumbnails/"))
		assert.Empty(t, listFiles(t, storage, "prints/"))
		runs, total, err := svc.List(t.Context(), domain.RunListFilter{
			Limit: 10,
		})
		require.NoError(t, err)
		require.Len(t, runs, 1)
		assert.Equal(t, 2, runs[0].Run)
		assert.Equal(t, 1, total)
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		err := svc.Delete(t.Context(), 9)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestGalleryService_Zip(t *testing.T) {
	t.Parallel()

	t.Run("returns an archive of the files", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		storage := newTestStorage(t)
		first := f.LoremIpsumSentence(6)
		second := f.LoremIpsumSentence(6)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte(first))
		putFile(t, storage, "captures/a/x_000003-02_20260101-000000.jpg", []byte(second))
		svc := newTestGalleryService(storage)
		files, err := svc.ListFiles(t.Context(), []int{3}, false)
		require.NoError(t, err)
		var buf bytes.Buffer

		err = svc.Zip(t.Context(), &buf, files)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"x_000003-01_20260101-000000.jpg": first,
			"x_000003-02_20260101-000000.jpg": second,
		}, zipContents(t, buf.Bytes()), "entries carry only the file name")
	})
}

// putGalleryEvents returns a storage with two runs of one event, a run of another event, and one original.
func putGalleryEvents(t *testing.T) media.Storage {
	t.Helper()

	storage := newTestStorage(t)
	putFile(t, storage, "captures/party/x_000002-01_20260101-000000.jpg", []byte("b"))
	putFile(t, storage, "captures/party/x_000001-01_20260101-000000.jpg", []byte("a"))
	putFile(t, storage, "captures/other/x_000003-01_20260101-000000.jpg", []byte("c"))
	putFile(t, storage, "originals/party/x_000001-01_20260101-000000.jpg", []byte("original"))
	return storage
}

// newTestGalleryService builds a gallery service on storage, with a capture service that stores nothing.
func newTestGalleryService(storage media.Storage) *GalleryService {
	runs, captures := newTestStores(storage)
	return NewGalleryService(runs, captures, newCaptureServiceOn(storage, runs, captures, captureConfig(1)))
}

// capturePaths returns the storage paths of files in their order.
func capturePaths(files []domain.CaptureFile) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

// zipContents reads every entry of a zip archive, keyed by its name.
func zipContents(t *testing.T, data []byte) map[string]string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	contents := map[string]string{}
	for _, file := range reader.File {
		rc, err := file.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		contents[file.Name] = string(content)
	}
	return contents
}
