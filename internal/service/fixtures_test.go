package service

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/store"
)

// newTestStorage returns a local storage in a temporary directory of the test.
func newTestStorage(t *testing.T) media.Storage {
	t.Helper()

	storage, err := media.New(media.Options{
		Driver:    media.DriverLocal,
		LocalPath: t.TempDir(),
	})
	require.NoError(t, err)
	return storage
}

// putFile stores data at path in storage.
func putFile(t *testing.T, storage media.Storage, storagePath string, data []byte) {
	t.Helper()

	require.NoError(t, storage.Put(t.Context(), storagePath, bytes.NewReader(data), int64(len(data))))
}

// readFile reads path from storage.
func readFile(t *testing.T, storage media.Storage, storagePath string) []byte {
	t.Helper()

	data, err := media.ReadAll(t.Context(), storage, storagePath)
	require.NoError(t, err)
	return data
}

// listFiles lists every file below prefix in storage.
func listFiles(t *testing.T, storage media.Storage, prefix string) []string {
	t.Helper()

	paths, err := storage.List(t.Context(), prefix)
	require.NoError(t, err)
	return paths
}

// testJPEG encodes a solid image of the given size as JPEG.
func testJPEG(t *testing.T, width, height int, c color.Color) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, solidImage(width, height, c), &jpeg.Options{
		Quality: 95,
	}))
	return buf.Bytes()
}

// solidImage returns an image of the given size filled with c.
func solidImage(width, height int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, c)
		}
	}
	return img
}

// testPNG encodes img as PNG.
func testPNG(t *testing.T, img image.Image) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// decodeJPEG decodes JPEG data and fails the test if it is not a JPEG.
func decodeJPEG(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := jpeg.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	return img
}

// pad writes a run number the way captures are stored.
func pad(run int) string {
	return fmt.Sprintf("%06d", run)
}

// captureConfig returns capture options with a fixed file prefix and event name, which the stored paths show.
func captureConfig(imagesPerRun int) CaptureOptions {
	return CaptureOptions{
		ImagesPerRun: imagesPerRun,
		FilePrefix:   "booth_",
		EventName:    "Summer Party",
	}
}

// pixel returns the color of img at x, y as RGBA.
func pixel(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// newCaptureServiceOn builds a capture service with the stores on storage.
func newCaptureServiceOn(storage media.Storage, cfg CaptureOptions) *CaptureService {
	return NewCaptureService(zerolog.Nop(), cfg, store.NewRunStore(storage), store.NewCaptureStore(storage), store.NewOverlayStore(storage))
}

// marshalLayout encodes layout as JSON.
func marshalLayout(t *testing.T, layout domain.Layout) []byte {
	t.Helper()

	data, err := json.Marshal(layout)
	require.NoError(t, err)
	return data
}

// assertValidation checks that err is a validation error with the tag on the field.
func assertValidation(t *testing.T, err error, field, tag string) {
	t.Helper()

	validation, ok := errors.AsType[*apperr.ValidationError](err)
	require.True(t, ok)
	require.Contains(t, validation.Fields, field)
	tags := make([]string, 0, len(validation.Fields[field]))
	for _, fieldErr := range validation.Fields[field] {
		tags = append(tags, fieldErr.Tag)
	}
	assert.Contains(t, tags, tag)
}
