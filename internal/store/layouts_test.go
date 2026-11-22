package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/domain"
)

func TestLayoutStore_List(t *testing.T) {
	t.Parallel()

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

	t.Run("rejects a broken file", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "layouts/broken.json", []byte("{"))

		_, err := NewLayoutStore(storage).Get(t.Context(), "broken")

		assert.ErrorContains(t, err, `decoding layout "broken"`)
	})
}

func TestLayoutStore_Seed(t *testing.T) {
	t.Parallel()

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
func testLayout(name string) domain.Layout {
	return domain.Layout{
		Name: name,
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
