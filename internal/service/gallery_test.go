package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/store"
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
			},
			{
				Run:        3,
				Event:      "event-1",
				ImageCount: 3,
			},
			{
				Run:        2,
				Event:      "event-0",
				ImageCount: 2,
			},
			{
				Run:        1,
				Event:      "event-1",
				ImageCount: 1,
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
					},
					{
						Run:        3,
						Event:      "event-1",
						ImageCount: 3,
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
					},
					{
						Run:        1,
						Event:      "event-1",
						ImageCount: 1,
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
				RunCount:      1,
				ImageCount:    1,
				OriginalCount: 1,
				LastRun:       2,
			},
			{
				Event:      "old-party",
				RunCount:   1,
				ImageCount: 2,
				LastRun:    1,
			},
		}, events, "the most recent event comes first, and the event of the booth is marked")
	})
}

func TestGalleryService_ListFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures of the run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-02_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000004-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		paths, err := svc.ListFiles(t.Context(), 3, false)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"captures/a/x_000003-01_20260101-000000.jpg",
			"captures/a/x_000003-02_20260101-000000.jpg",
		}, paths)
	})

	t.Run("returns the originals", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("composited"))
		putFile(t, storage, "originals/a/x_000003-01_20260101-000000.jpg", []byte("original"))
		svc := newTestGalleryService(storage)

		paths, err := svc.ListFiles(t.Context(), 3, true)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"originals/a/x_000003-01_20260101-000000.jpg",
		}, paths)
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		_, err := svc.ListFiles(t.Context(), 4, false)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for a run without originals", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		_, err := svc.ListFiles(t.Context(), 3, true)

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

		data, err := svc.GetCapture(t.Context(), 3, 1, false)

		require.NoError(t, err)
		assert.Equal(t, []byte("composited"), data)
	})

	t.Run("returns the original", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("composited"))
		putFile(t, storage, "originals/a/x_000003-01_20260101-000000.jpg", []byte("original"))
		svc := newTestGalleryService(storage)

		data, err := svc.GetCapture(t.Context(), 3, 1, true)

		require.NoError(t, err)
		assert.Equal(t, []byte("original"), data)
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

func TestGalleryService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000001-02_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "prints/a/x_000001-collage-1_0a1b2c3d.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		err := svc.Delete(t.Context(), 1)

		require.NoError(t, err)
		assert.Equal(t, []string{"captures/a/x_000002-01_20260101-000000.jpg"}, listFiles(t, storage, "captures/"))
		assert.Empty(t, listFiles(t, storage, "originals/"))
		assert.Empty(t, listFiles(t, storage, "prints/"))
	})

	t.Run("accepts an unknown run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))
		svc := newTestGalleryService(storage)

		err := svc.Delete(t.Context(), 9)

		assert.NoError(t, err, "deleting an unknown run is not an error")
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
		paths, err := svc.ListFiles(t.Context(), 3, false)
		require.NoError(t, err)
		var buf bytes.Buffer

		err = svc.Zip(t.Context(), &buf, paths)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"x_000003-01_20260101-000000.jpg": first,
			"x_000003-02_20260101-000000.jpg": second,
		}, zipContents(t, buf.Bytes()), "entries carry only the file name")
	})
}

func TestGalleryService_ZipEvent(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures of the event ordered by run", func(t *testing.T) {
		t.Parallel()

		svc := newTestGalleryService(putGalleryEvents(t))
		var buf bytes.Buffer

		err := svc.ZipEvent(t.Context(), &buf, "party", false)

		require.NoError(t, err)
		reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)
		names := make([]string, 0, len(reader.File))
		for _, file := range reader.File {
			names = append(names, file.Name)
		}
		assert.Equal(t, []string{
			"x_000001-01_20260101-000000.jpg",
			"x_000002-01_20260101-000000.jpg",
		}, names, "every capture of the event, ordered by run")
	})

	t.Run("returns the originals", func(t *testing.T) {
		t.Parallel()

		svc := newTestGalleryService(putGalleryEvents(t))
		var buf bytes.Buffer

		err := svc.ZipEvent(t.Context(), &buf, "party", true)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"x_000001-01_20260101-000000.jpg": "original",
		}, zipContents(t, buf.Bytes()))
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
	return NewGalleryService(store.NewRunStore(storage), store.NewCaptureStore(storage), newCaptureServiceOn(storage, captureConfig(1)))
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
