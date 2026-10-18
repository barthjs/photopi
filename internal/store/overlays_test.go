package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
)

func TestOverlayStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the overlays sorted by name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		f := apitest.Faker(t)
		st := NewOverlayStore(newTestStorage(t))
		_, err := st.Create(ctx, "zeta.png", []byte(f.LoremIpsumSentence(6)))
		require.NoError(t, err)
		_, err = st.Create(ctx, "alpha.png", []byte(f.LoremIpsumSentence(6)))
		require.NoError(t, err)

		overlays, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Overlay{
			{
				Name: "alpha.png",
				Path: "overlays/alpha.png",
			},
			{
				Name: "zeta.png",
				Path: "overlays/zeta.png",
			},
		}, overlays)
	})

	t.Run("returns no overlays without the folder", func(t *testing.T) {
		t.Parallel()

		overlays, err := NewOverlayStore(newTestStorage(t)).List(t.Context())

		require.NoError(t, err)
		assert.Empty(t, overlays)
	})

	t.Run("skips other files", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage := newTestStorage(t)
		st := NewOverlayStore(storage)
		_, err := st.Create(ctx, "valid.png", []byte("x"))
		require.NoError(t, err)
		putFile(t, storage, "overlays/notes.txt", []byte{})
		putFile(t, storage, "overlays/nested/deep.png", []byte("x"))
		putFile(t, storage, "other.png", []byte("x"))

		overlays, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Overlay{
			{
				Name: "valid.png",
				Path: "overlays/valid.png",
			},
		}, overlays)
	})
}

func TestOverlayStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the overlay", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewOverlayStore(newTestStorage(t))
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))
		overlay, err := st.Create(ctx, "alpha.png", content)
		require.NoError(t, err)

		data, err := st.Get(ctx, overlay.Path)

		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("returns not found for an unknown overlay", func(t *testing.T) {
		t.Parallel()

		_, err := NewOverlayStore(newTestStorage(t)).Get(t.Context(), "overlays/missing.png")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestOverlayStore_Create(t *testing.T) {
	t.Parallel()

	t.Run("stores the overlay", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))

		overlay, err := NewOverlayStore(storage).Create(t.Context(), "alpha.png", content)

		require.NoError(t, err)
		assert.Equal(t, domain.Overlay{
			Name: "alpha.png",
			Path: "overlays/alpha.png",
		}, overlay)
		assert.Equal(t, content, readFile(t, storage, "overlays/alpha.png"))
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewOverlayStore(storage)

		for _, name := range invalidOverlayNames() {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := st.Create(t.Context(), name, []byte("x"))

				assert.ErrorIs(t, err, ErrInvalidName)
				assert.Empty(t, listFiles(t, storage, ""))
			})
		}
	})
}

func TestOverlayStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the overlay", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage := newTestStorage(t)
		st := NewOverlayStore(storage)
		_, err := st.Create(ctx, "alpha.png", []byte("x"))
		require.NoError(t, err)

		err = st.Delete(ctx, "alpha.png")

		require.NoError(t, err)
		assert.Empty(t, listFiles(t, storage, "overlays/"))
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewOverlayStore(storage)
		_, err := st.Create(t.Context(), "alpha.png", []byte("x"))
		require.NoError(t, err)

		for _, name := range invalidOverlayNames() {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				err := st.Delete(t.Context(), name)

				assert.ErrorIs(t, err, ErrInvalidName)
				assert.Equal(t, []string{"overlays/alpha.png"}, listFiles(t, storage, "overlays/"))
			})
		}
	})

	t.Run("returns not found for an unknown overlay", func(t *testing.T) {
		t.Parallel()

		err := NewOverlayStore(newTestStorage(t)).Delete(t.Context(), "missing.png")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestOverlayStore_Path(t *testing.T) {
	t.Parallel()

	t.Run("returns the file of the overlay", func(t *testing.T) {
		t.Parallel()

		p, err := NewOverlayStore(newTestStorage(t)).Path("alpha.png")

		require.NoError(t, err)
		assert.Equal(t, "overlays/alpha.png", p)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewOverlayStore(newTestStorage(t))

		for _, name := range invalidOverlayNames() {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := st.Path(name)

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})
}

// invalidOverlayNames returns names that cannot address an overlay file,
// the invalid file names with the overlay extension and names without it.
func invalidOverlayNames() []string {
	names := []string{"", "alpha", "alpha.jpg"}
	for _, name := range invalidFileNames(overlayExtension) {
		names = append(names, name+overlayExtension)
	}
	return names
}
