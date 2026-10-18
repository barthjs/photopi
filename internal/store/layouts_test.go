package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
)

func TestLayoutStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the layouts sorted by id", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewLayoutStore(newTestStorage(t))
		require.NoError(t, st.Save(ctx, "b", testLayout("b")))
		require.NoError(t, st.Save(ctx, "a", testLayout("a")))

		layouts, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Layout{
			testLayout("a"),
			testLayout("b"),
		}, layouts, "sorted by ID")
	})

	t.Run("returns no layouts without the folder", func(t *testing.T) {
		t.Parallel()

		layouts, err := NewLayoutStore(newTestStorage(t)).List(t.Context())

		require.NoError(t, err)
		assert.Empty(t, layouts)
	})

	t.Run("skips other files", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage := newTestStorage(t)
		st := NewLayoutStore(storage)
		require.NoError(t, st.Save(ctx, "valid", testLayout("valid")))
		putFile(t, storage, "layouts/notes.txt", []byte{})
		putFile(t, storage, "layouts/nested/deep.json", []byte("{}"))
		putFile(t, storage, "other.json", []byte("{}"))

		layouts, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Layout{
			testLayout("valid"),
		}, layouts)
	})

	t.Run("rejects a broken file", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "layouts/broken.json", []byte("{"))

		_, err := NewLayoutStore(storage).List(t.Context())

		assert.ErrorContains(t, err, "decoding layout")
	})
}

func TestLayoutStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the layout", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewLayoutStore(newTestStorage(t))
		require.NoError(t, st.Save(ctx, "b", testLayout("b")))

		got, err := st.Get(ctx, "b")

		require.NoError(t, err)
		assert.Equal(t, new(testLayout("b")), got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewLayoutStore(newTestStorage(t))

		for _, id := range invalidFileNames(layoutExtension) {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				_, err := st.Get(t.Context(), id)

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()

		_, err := NewLayoutStore(newTestStorage(t)).Get(t.Context(), "missing")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("rejects a broken file", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "layouts/broken.json", []byte("{"))

		_, err := NewLayoutStore(storage).Get(t.Context(), "broken")

		assert.ErrorContains(t, err, `decoding layout "broken"`)
	})
}

func TestLayoutStore_Save(t *testing.T) {
	t.Parallel()

	t.Run("stores the layout", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)

		err := NewLayoutStore(storage).Save(t.Context(), "b", testLayout("b"))

		require.NoError(t, err)
		assert.Equal(t, []string{"layouts/b.json"}, listFiles(t, storage, "layouts/"))
	})

	t.Run("replaces the layout", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewLayoutStore(newTestStorage(t))
		require.NoError(t, st.Save(ctx, "b", testLayout("b")))
		changed := testLayout("b")
		changed.Name = "Changed"

		err := st.Save(ctx, "b", changed)

		require.NoError(t, err)
		got, err := st.Get(ctx, "b")
		require.NoError(t, err)
		assert.Equal(t, &changed, got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewLayoutStore(storage)

		for _, id := range invalidFileNames(layoutExtension) {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				err := st.Save(t.Context(), id, testLayout(id))

				assert.ErrorIs(t, err, ErrInvalidName)
				assert.Empty(t, listFiles(t, storage, ""))
			})
		}
	})
}

func TestLayoutStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the layout", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewLayoutStore(newTestStorage(t))
		require.NoError(t, st.Save(ctx, "b", testLayout("b")))

		err := st.Delete(ctx, "b")

		require.NoError(t, err)
		_, err = st.Get(ctx, "b")
		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewLayoutStore(newTestStorage(t))

		for _, id := range invalidFileNames(layoutExtension) {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				err := st.Delete(t.Context(), id)

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()

		err := NewLayoutStore(newTestStorage(t)).Delete(t.Context(), "missing")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestLayoutStore_Seed(t *testing.T) {
	t.Parallel()

	t.Run("stores the layouts", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewLayoutStore(newTestStorage(t))

		err := st.Seed(ctx, []domain.Layout{
			testLayout("a"),
			testLayout("b"),
		})

		require.NoError(t, err)
		layouts, err := st.List(ctx)
		require.NoError(t, err)
		assert.Equal(t, []domain.Layout{
			testLayout("a"),
			testLayout("b"),
		}, layouts)
	})

	t.Run("leaves existing layouts alone", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "layouts/own.json", []byte("{}"))

		err := NewLayoutStore(storage).Seed(t.Context(), []domain.Layout{
			testLayout("a"),
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"layouts/own.json"}, listFiles(t, storage, "layouts/"))
	})
}

// testLayout returns a valid layout with one locked full-page slot.
func testLayout(id string) domain.Layout {
	return domain.Layout{
		ID:   id,
		Name: "Test " + id,
		Paper: domain.LayoutPaper{
			WidthMM:  100,
			HeightMM: 150,
		},
		Pages: []domain.LayoutPage{
			{
				Slots: []domain.LayoutSlot{
					{
						X:      0,
						Y:      0,
						W:      1,
						H:      1,
						Locked: true,
					},
				},
			},
		},
	}
}
