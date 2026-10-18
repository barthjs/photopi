package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
)

func TestCaptureStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures of every run and event", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/b/x_000002-03_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000001-01_20260101-000000.jpg", []byte("x"))

		captures, err := NewCaptureStore(storage).List(t.Context(), false)

		require.NoError(t, err)
		assert.ElementsMatch(t, []domain.CaptureFile{
			{
				Path:       "captures/a/x_000001-01_20260101-000000.jpg",
				Event:      "a",
				Run:        1,
				ImageIndex: 1,
			},
			{
				Path:       "captures/b/x_000002-03_20260101-000000.jpg",
				Event:      "b",
				Run:        2,
				ImageIndex: 3,
			},
		}, captures)
	})

	t.Run("returns the originals", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000001-01_20260101-000000.jpg", []byte("x"))

		captures, err := NewCaptureStore(storage).List(t.Context(), true)

		require.NoError(t, err)
		assert.Equal(t, []domain.CaptureFile{
			{
				Path:       "originals/a/x_000001-01_20260101-000000.jpg",
				Event:      "a",
				Run:        1,
				ImageIndex: 1,
			},
		}, captures)
	})

	t.Run("skips files that are no capture", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000002-collage-1_0a1b2c3d.jpg", []byte("x"))

		captures, err := NewCaptureStore(storage).List(t.Context(), false)

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

		paths, err := NewCaptureStore(storage).ListByRun(t.Context(), 2, false)

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

		paths, err := NewCaptureStore(storage).ListByRun(t.Context(), 2, true)

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

		paths, err := NewCaptureStore(storage).ListByRun(t.Context(), 2, false)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"captures/a/x_000002-01_20260101-000000.jpg",
		}, paths)
	})

	t.Run("returns no captures for an unknown run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))

		paths, err := NewCaptureStore(storage).ListByRun(t.Context(), 3, false)

		require.NoError(t, err)
		assert.Empty(t, paths)
	})
}

func TestCaptureStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the capture", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))
		putFile(t, storage, "captures/a/x_000004-02_20260101-000000.jpg", content)

		data, err := NewCaptureStore(storage).Get(t.Context(), 4, 2, false)

		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("returns the original", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))
		putFile(t, storage, "captures/a/x_000004-02_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000004-02_20260101-000000.jpg", content)

		data, err := NewCaptureStore(storage).Get(t.Context(), 4, 2, true)

		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("returns not found for an unknown image", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000004-02_20260101-000000.jpg", []byte("x"))

		_, err := NewCaptureStore(storage).Get(t.Context(), 4, 1, false)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestCaptureStore_Create(t *testing.T) {
	t.Parallel()

	capture := domain.Capture{
		Event:      "party",
		Prefix:     "booth_",
		Run:        7,
		ImageIndex: 2,
		TakenAt:    time.Date(2026, 9, 6, 14, 4, 23, 0, time.UTC),
	}

	t.Run("stores the capture in the folder of its event", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))

		err := NewCaptureStore(storage).Create(t.Context(), capture, content, nil)

		require.NoError(t, err)
		assert.Equal(t, content, readFile(t, storage, "captures/party/booth_000007-02_20260906-140423.jpg"))
		assert.Empty(t, listFiles(t, storage, "originals/"), "without an original, none is stored")
	})

	t.Run("stores the original under the same name", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		original := []byte(apitest.Faker(t).LoremIpsumSentence(6))

		err := NewCaptureStore(storage).Create(t.Context(), capture, []byte("x"), original)

		require.NoError(t, err)
		assert.Equal(t, original, readFile(t, storage, "originals/party/booth_000007-02_20260906-140423.jpg"))
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

	t.Run("removes the capture and its original", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000004-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000004-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000004-02_20260101-000000.jpg", []byte("x"))

		err := NewCaptureStore(storage).Delete(t.Context(), 4, 1)

		require.NoError(t, err)
		assert.Equal(t, []string{"captures/a/x_000004-02_20260101-000000.jpg"}, listFiles(t, storage, "captures/"))
		assert.Empty(t, listFiles(t, storage, "originals/"))
	})

	t.Run("accepts an image without files", func(t *testing.T) {
		t.Parallel()

		err := NewCaptureStore(newTestStorage(t)).Delete(t.Context(), 4, 1)

		assert.NoError(t, err)
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
