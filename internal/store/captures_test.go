package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/domain"
)

func TestCaptureStore_List(t *testing.T) {
	t.Parallel()

	t.Run("skips files that are no capture", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000002-collage-1_0a1b2c3d.jpg", []byte("x"))

		captures, err := newTestCaptureStore(storage).List(t.Context(), false)

		require.NoError(t, err)
		assert.Empty(t, captures)
	})
}

func TestCaptureStore_ListByRun(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures of the run ordered by index", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		// The clock went back between the shots, the image index still decides the order
		putFile(t, storage, "captures/a/x_000002-02_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000002-01_20260101-120000.jpg", []byte("x"))
		putFile(t, storage, "captures/b/x_000002-10_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte("x"))

		paths, err := newTestCaptureStore(storage).ListByRun(t.Context(), 2, false)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"captures/a/x_000002-01_20260101-120000.jpg",
			"captures/a/x_000002-02_20260101-000000.jpg",
			"captures/b/x_000002-10_20260101-000000.jpg",
		}, paths)
	})

	t.Run("returns the originals of the run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000002-01_20260101-000000.jpg", []byte("x"))

		paths, err := newTestCaptureStore(storage).ListByRun(t.Context(), 2, true)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"originals/a/x_000002-01_20260101-000000.jpg",
		}, paths)
	})

	t.Run("skips files that are no capture", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000002-collage-1_0a1b2c3d.jpg", []byte("x"))

		paths, err := newTestCaptureStore(storage).ListByRun(t.Context(), 2, false)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"captures/a/x_000002-01_20260101-000000.jpg",
		}, paths)
	})
}

func TestCaptureFileName(t *testing.T) {
	t.Parallel()

	t.Run("names the file after the run, the image, and the time it was taken", func(t *testing.T) {
		t.Parallel()

		name := captureFileName(domain.Capture{
			Event:      "party",
			Prefix:     "Party",
			Run:        11,
			ImageIndex: 1,
			TakenAt:    time.Date(2026, 9, 6, 14, 4, 23, 0, time.UTC),
		})

		assert.Equal(t, "Party_000011-01_20260906-140423.jpg", name)
	})
}

func TestCaptureStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the capture, its original, and its thumbnail from the storage and the listing", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000004-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000004-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "thumbnails/a/x_000004-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000004-02_20260101-000000.jpg", []byte("x"))
		st := newTestCaptureStore(storage)
		_, err := st.List(t.Context(), false)
		require.NoError(t, err)

		err = st.Delete(t.Context(), 4, 1)

		require.NoError(t, err)
		assert.Equal(t, []string{"captures/a/x_000004-02_20260101-000000.jpg"}, listFiles(t, storage, "captures/"))
		assert.Empty(t, listFiles(t, storage, "originals/"))
		assert.Empty(t, listFiles(t, storage, "thumbnails/"))
		captures, err := st.List(t.Context(), false)
		require.NoError(t, err)
		require.Len(t, captures, 1)
		assert.Equal(t, 2, captures[0].ImageIndex)
	})

	t.Run("accepts an image without files", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)

		err := newTestCaptureStore(storage).Delete(t.Context(), 4, 1)

		assert.NoError(t, err)
	})
}

func TestParseTakenAt(t *testing.T) {
	t.Parallel()

	t.Run("reads the time in the local time zone", func(t *testing.T) {
		t.Parallel()

		takenAt := parseTakenAt("20260906-140423")

		assert.Equal(t, time.Date(2026, 9, 6, 14, 4, 23, 0, time.Local), takenAt)
	})

	t.Run("returns zero for a suffix that is no time", func(t *testing.T) {
		t.Parallel()

		takenAt := parseTakenAt(apitest.Faker(t).LetterN(8))

		assert.True(t, takenAt.IsZero())
	})
}

func TestCapturePart(t *testing.T) {
	t.Parallel()

	t.Run("pads the image index to two digits", func(t *testing.T) {
		t.Parallel()

		part := capturePart(3)

		assert.Equal(t, "03", part)
	})
}

func TestCaptureFolder(t *testing.T) {
	t.Parallel()

	t.Run("returns the folder of the captures", func(t *testing.T) {
		t.Parallel()

		folder := captureFolder(false)

		assert.Equal(t, capturesFolder, folder)
	})

	t.Run("returns the folder of the originals", func(t *testing.T) {
		t.Parallel()

		folder := captureFolder(true)

		assert.Equal(t, originalsFolder, folder)
	})
}
