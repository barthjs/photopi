package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
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
