package store

import (
	"archive/zip"
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
)

func TestRunStore_GetFile(t *testing.T) {
	t.Parallel()

	t.Run("returns the file", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))
		putFile(t, storage, "prints/a/x_000001-collage-1_0a1b2c3d.jpg", content)

		data, err := NewRunStore(storage).GetFile(t.Context(), "prints/a/x_000001-collage-1_0a1b2c3d.jpg")

		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("returns not found for an unknown file", func(t *testing.T) {
		t.Parallel()

		st := NewRunStore(newTestStorage(t))

		_, err := st.GetFile(t.Context(), "captures/a/x_000001-01_20260101-000000.jpg")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRunStore_Create(t *testing.T) {
	t.Parallel()

	t.Run("starts at one", func(t *testing.T) {
		t.Parallel()

		st := NewRunStore(newTestStorage(t))

		next, err := st.Create(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 1, next)
	})

	t.Run("continues after the highest stored run without a counter", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/earlier-event/x_000041-01_20260101-000000.jpg", []byte("x"))

		next, err := NewRunStore(storage).Create(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 42, next)
	})

	t.Run("never hands out a number twice, even after the run was deleted", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage := newTestStorage(t)
		st := NewRunStore(storage)
		first, err := st.Create(ctx)
		require.NoError(t, err)
		putFile(t, storage, fmt.Sprintf("captures/a/x_%06d-01_20260101-000000.jpg", first), []byte("x"))
		require.NoError(t, st.Delete(ctx, first))

		second, err := st.Create(ctx)

		require.NoError(t, err)
		assert.Equal(t, first+1, second)
	})

	t.Run("replaces a damaged counter", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, runCounterPath, []byte("garbage"))
		putFile(t, storage, "captures/a/x_000005-01_20260101-000000.jpg", []byte("x"))

		next, err := NewRunStore(storage).Create(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 6, next)
	})

	t.Run("keeps the counter out of the run files", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)

		_, err := NewRunStore(storage).Create(t.Context())

		require.NoError(t, err)
		assert.Empty(t, listFiles(t, storage, "captures/"))
	})
}

func TestRunStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the run", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "originals/a/x_000001-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "prints/a/x_000001-collage-1_0a1b2c3d.jpg", []byte("x"))
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))

		err := NewRunStore(storage).Delete(t.Context(), 1)

		require.NoError(t, err)
		assert.Equal(t, []string{"captures/a/x_000002-01_20260101-000000.jpg"}, listFiles(t, storage, "captures/"))
		assert.Empty(t, listFiles(t, storage, "originals/"))
		assert.Empty(t, listFiles(t, storage, "prints/"))
	})

	t.Run("accepts a run without files", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))

		err := NewRunStore(storage).Delete(t.Context(), 99)

		assert.NoError(t, err, "a run without files is not an error")
	})
}

func TestRunStore_SignFile(t *testing.T) {
	t.Parallel()

	t.Run("returns a signed URL of the file", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/a/x_000001-01_20260101-000000.jpg", []byte("x"))

		url, err := NewRunStore(storage).SignFile(t.Context(), "captures/a/x_000001-01_20260101-000000.jpg", time.Hour)

		require.NoError(t, err)
		assert.Contains(t, url, "x_000001-01_20260101-000000.jpg")
		assert.Contains(t, url, "signature=")
	})

	t.Run("rejects a lifetime that is not positive", func(t *testing.T) {
		t.Parallel()

		st := NewRunStore(newTestStorage(t))

		_, err := st.SignFile(t.Context(), "captures/a/x_000001-01_20260101-000000.jpg", 0)

		assert.Error(t, err)
	})
}

func TestRunStore_ZipFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns an archive of the files", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		storage := newTestStorage(t)
		first := f.LoremIpsumSentence(6)
		second := f.LoremIpsumSentence(6)
		putFile(t, storage, "captures/a/x_000003-01_20260101-000000.jpg", []byte(first))
		putFile(t, storage, "prints/a/x_000003-collage-1_0a1b2c3d.jpg", []byte(second))
		var buf bytes.Buffer

		err := NewRunStore(storage).ZipFiles(t.Context(), &buf, []string{
			"captures/a/x_000003-01_20260101-000000.jpg",
			"prints/a/x_000003-collage-1_0a1b2c3d.jpg",
		})

		require.NoError(t, err)
		reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)
		for _, file := range reader.File {
			assert.Equal(t, zip.Store, file.Method, "JPEGs are stored without compression")
		}
		assert.Equal(t, map[string]string{
			"x_000003-01_20260101-000000.jpg": first,
			"x_000003-collage-1_0a1b2c3d.jpg": second,
		}, zipContents(t, buf.Bytes()), "entries carry only the file name")
	})

	t.Run("returns an empty archive without files", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := NewRunStore(newTestStorage(t)).ZipFiles(t.Context(), &buf, nil)

		require.NoError(t, err)
		assert.Empty(t, zipContents(t, buf.Bytes()))
	})

	t.Run("rejects a missing file", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := NewRunStore(newTestStorage(t)).ZipFiles(t.Context(), &buf, []string{
			"captures/a/x_000003-01_20260101-000000.jpg",
		})

		assert.ErrorContains(t, err, "x_000003-01_20260101-000000.jpg")
	})
}

func TestRunFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{
			name:   "builds the name from a plain prefix",
			prefix: "Party",
			want:   "Party_000011-01_20260906-140423.jpg",
		},
		{
			name:   "keeps one separator after a prefix ending with it",
			prefix: "Party_",
			want:   "Party_000011-01_20260906-140423.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name := runFileName(tt.prefix, 11, "01", "20260906-140423")

			assert.Equal(t, tt.want, name)
		})
	}
}

func TestListRunFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns only the files directly inside an event folder", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "captures/old-event/a_000009-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/new-event/a_000003-02_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/a_000004-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/new-event/nested/a_000005-01_20260101-000000.jpg", []byte("x"))
		putFile(t, storage, "captures/new-event/notes.txt", []byte("x"))

		files, err := listRunFiles(t.Context(), storage, capturesFolder)

		require.NoError(t, err)
		assert.ElementsMatch(t, []runFile{
			{
				path:  "captures/old-event/a_000009-01_20260101-000000.jpg",
				event: "old-event",
				run:   9,
				part:  "01",
			},
			{
				path:  "captures/new-event/a_000003-02_20260101-000000.jpg",
				event: "new-event",
				run:   3,
				part:  "02",
			},
		}, files, "only the files directly inside an event folder belong to runs")
	})
}

func TestParseRunFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     string
		wantRun  int
		wantPart string
		wantOK   bool
	}{
		{
			name:     "parses a capture",
			file:     "Vivi+Meiky_000011-04_20260906-140507.jpg",
			wantRun:  11,
			wantPart: "04",
			wantOK:   true,
		},
		{
			name:     "parses a collage page",
			file:     "Party_000003-collage-2_0a1b2c3d.jpg",
			wantRun:  3,
			wantPart: "collage-2",
			wantOK:   true,
		},
		{
			name:     "parses a prefix with underscores",
			file:     "my_party_000007-01_20260101-000000.jpg",
			wantRun:  7,
			wantPart: "01",
			wantOK:   true,
		},
		{
			name:   "rejects a file of another layout",
			file:   "photo_01.jpg",
			wantOK: false,
		},
		{
			name:   "rejects a file that is no jpeg",
			file:   "Party_000003-01_20260101-000000.png",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			run, part, ok := parseRunFile(tt.file)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantRun, run)
			assert.Equal(t, tt.wantPart, part)
		})
	}
}
