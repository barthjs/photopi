package service

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/media"
)

func TestCaptureService_ListRunFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns the files of the run", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(3), "")
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))

		paths, err := svc.ListRunFiles(t.Context(), 2)

		require.NoError(t, err)
		assert.Equal(t, []string{"captures/a/x_000002-01_20260101-000000.jpg"}, paths)
	})

	t.Run("waits for the capture being stored", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(2), "")
		run := startRun(t, svc)
		capture := testJPEG(t, 8, 8, color.White)
		_, err := svc.Create(t.Context(), run, &capture)
		require.NoError(t, err)

		paths, err := svc.ListRunFiles(t.Context(), run)

		require.NoError(t, err)
		assert.Len(t, paths, 1)
	})

	t.Run("returns no files for an unknown run", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(3), "")
		putFile(t, storage, "captures/a/x_000002-01_20260101-000000.jpg", []byte("x"))

		paths, err := svc.ListRunFiles(t.Context(), 3)

		require.NoError(t, err)
		assert.Empty(t, paths)
	})
}

func TestCaptureService_Get(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*CaptureService, []byte) {
		t.Helper()

		f := apitest.Faker(t)
		svc, storage := newTestCaptureService(t, captureConfig(3), "")
		second := []byte(f.LoremIpsumSentence(6))
		putFile(t, storage, "captures/old-event/booth_000005-01_20260101-120000.jpg", []byte(f.LoremIpsumSentence(6)))
		putFile(t, storage, "captures/old-event/booth_000005-02_20260101-120005.jpg", second)
		return svc, second
	}

	t.Run("returns the capture", func(t *testing.T) {
		t.Parallel()

		svc, second := setup(t)

		data, err := svc.Get(t.Context(), 5, 2)

		require.NoError(t, err)
		assert.Equal(t, second, data)
	})

	t.Run("waits for the capture being stored", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(2), "")
		run := startRun(t, svc)
		capture := testJPEG(t, 8, 8, color.White)
		_, err := svc.Create(t.Context(), run, &capture)
		require.NoError(t, err)

		data, err := svc.Get(t.Context(), run, 1)

		require.NoError(t, err)
		assert.Equal(t, capture, data)
	})

	t.Run("returns not found for an unknown image", func(t *testing.T) {
		t.Parallel()

		svc, _ := setup(t)

		_, err := svc.Get(t.Context(), 5, 3)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		svc, _ := setup(t)

		_, err := svc.Get(t.Context(), 6, 1)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestCaptureService_GetPreview(t *testing.T) {
	t.Parallel()

	t.Run("returns the screen version of a capture right after the shot", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(2), "")
		run := startRun(t, svc)
		capture := testJPEG(t, 1600, 960, color.White)
		_, err := svc.Create(t.Context(), run, &capture)
		require.NoError(t, err)

		preview, err := svc.GetPreview(t.Context(), run, 1)

		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, 800, 480), decodeJPEG(t, preview).Bounds(), "the screen version fits the kiosk")
	})

	t.Run("draws the overlay into the screen version", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(2), "overlays/frame.png")
		overlay := image.NewRGBA(image.Rect(0, 0, 2, 2))
		for x := range 2 {
			for y := range 2 {
				overlay.Set(x, y, color.RGBA{
					R: 255,
					A: 255,
				})
			}
		}
		putFile(t, storage, "overlays/frame.png", testPNG(t, overlay))
		run := startRun(t, svc)
		capture := testJPEG(t, 80, 40, color.White)
		_, err := svc.Create(t.Context(), run, &capture)
		require.NoError(t, err)

		preview, err := svc.GetPreview(t.Context(), run, 1)

		require.NoError(t, err)
		got := pixel(decodeJPEG(t, preview), 10, 10)
		assert.Greater(t, int(got.R)-int(got.G), 150)
	})

	t.Run("renders the screen version of a stored capture without one in memory", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(3), "")
		putFile(t, storage, "captures/a/x_000005-01_20260101-000000.jpg", testJPEG(t, 1600, 960, color.White))

		preview, err := svc.GetPreview(t.Context(), 5, 1)

		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, 800, 480), decodeJPEG(t, preview).Bounds())
	})

	t.Run("returns not found for an unknown capture", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(3), "")
		putFile(t, storage, "captures/a/x_000005-01_20260101-000000.jpg", testJPEG(t, 1600, 960, color.White))

		_, err := svc.GetPreview(t.Context(), 5, 2)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestCaptureService_Create(t *testing.T) {
	t.Parallel()

	// The overlay covers the left half in opaque red and leaves the right half transparent
	overlay := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for _, point := range []image.Point{
		{
			X: 0,
			Y: 0,
		},
		{
			X: 1,
			Y: 0,
		},
		{
			X: 0,
			Y: 1,
		},
		{
			X: 1,
			Y: 1,
		},
	} {
		overlay.Set(point.X, point.Y, color.RGBA{
			R: 255,
			A: 255,
		})
	}

	withOverlay := func(t *testing.T, saveOriginals bool) (*CaptureService, media.Storage, int) {
		t.Helper()

		cfg := captureConfig(2)
		cfg.SaveOriginals = saveOriginals
		svc, storage := newTestCaptureService(t, cfg, "overlays/frame.png")
		putFile(t, storage, "overlays/frame.png", testPNG(t, overlay))
		return svc, storage, startRun(t, svc)
	}

	t.Run("stores the captures flat in the folder of the event", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(2), "")
		jpegData := testJPEG(t, 8, 8, color.White)
		run := startRun(t, svc)
		first, err := svc.Create(t.Context(), run, &jpegData)
		require.NoError(t, err)

		second, err := svc.Create(t.Context(), run, &jpegData)

		require.NoError(t, err)
		assert.Equal(t, 1, first.ImageIndex)
		assert.Equal(t, 2, second.ImageIndex)
		assert.Equal(t, 2, second.ImagesPerRun)
		svc.Wait()
		paths := listFiles(t, storage, "captures/summer-party/")
		require.Len(t, paths, 2)
		for i, p := range paths {
			assert.True(t, strings.HasPrefix(p, fmt.Sprintf("captures/summer-party/booth_000001-%02d_", i+1)), p)
		}
		assert.Equal(t, jpegData, readFile(t, storage, paths[0]), "without an overlay, the capture is stored unchanged")
	})

	t.Run("replaces what a failed attempt left behind", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(2), "")
		run := startRun(t, svc)
		putFile(t, storage, "captures/summer-party/booth_"+pad(run)+"-01_20200101-000000.jpg", []byte("x"))
		jpegData := testJPEG(t, 8, 8, color.White)

		_, err := svc.Create(t.Context(), run, &jpegData)

		require.NoError(t, err)
		svc.Wait()
		paths := listFiles(t, storage, "captures/")
		require.Len(t, paths, 1)
		assert.Equal(t, jpegData, readFile(t, storage, paths[0]))
	})

	t.Run("composites the overlay scaled to the capture", func(t *testing.T) {
		t.Parallel()

		svc, storage, run := withOverlay(t, false)
		capture := testJPEG(t, 80, 40, color.White)

		_, err := svc.Create(t.Context(), run, &capture)

		require.NoError(t, err)
		svc.Wait()
		paths := listFiles(t, storage, "captures/")
		require.Len(t, paths, 1)
		img := decodeJPEG(t, readFile(t, storage, paths[0]))
		assert.Equal(t, image.Rect(0, 0, 80, 40), img.Bounds(), "the capture keeps its size")
		left := pixel(img, 10, 20)
		right := pixel(img, 70, 20)
		assert.Greater(t, int(left.R)-int(left.G), 150, "the opaque part of the overlay covers the capture")
		assert.Less(t, int(right.R)-int(right.G), 20, "the transparent part leaves the capture visible")
		assert.Empty(t, listFiles(t, storage, "originals/"))
	})

	t.Run("keeps applying the overlay to further captures", func(t *testing.T) {
		t.Parallel()

		svc, storage, run := withOverlay(t, false)
		capture := testJPEG(t, 80, 40, color.White)
		_, err := svc.Create(t.Context(), run, &capture)
		require.NoError(t, err)

		_, err = svc.Create(t.Context(), run, &capture)

		require.NoError(t, err)
		svc.Wait()
		paths := listFiles(t, storage, "captures/")
		require.Len(t, paths, 2)
		img := decodeJPEG(t, readFile(t, storage, paths[1]))
		left := pixel(img, 10, 20)
		assert.Greater(t, int(left.R)-int(left.G), 150, "the cached overlay is drawn as well")
	})

	t.Run("keeps the camera metadata with an overlay", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(1), "overlays/frame.png")
		putFile(t, storage, "overlays/frame.png", testPNG(t, image.NewRGBA(image.Rect(0, 0, 2, 2))))
		run := startRun(t, svc)
		exif := append([]byte{0xFF, 0xE1, 0x00, 0x0E}, []byte("Exif\x00\x00camera")...)
		plain := testJPEG(t, 8, 8, color.White)
		capture := append(append(append([]byte{}, plain[:2]...), exif...), plain[2:]...)

		_, err := svc.Create(t.Context(), run, &capture)

		require.NoError(t, err)
		svc.Wait()
		paths := listFiles(t, storage, "captures/")
		require.Len(t, paths, 1)
		stored := readFile(t, storage, paths[0])
		assert.Equal(t, exif, exifSegments(stored), "the composited capture keeps the camera metadata")
		decodeJPEG(t, stored)
	})

	t.Run("keeps the original when configured", func(t *testing.T) {
		t.Parallel()

		svc, storage, run := withOverlay(t, true)
		capture := testJPEG(t, 80, 40, color.White)

		_, err := svc.Create(t.Context(), run, &capture)

		require.NoError(t, err)
		svc.Wait()
		paths := listFiles(t, storage, "captures/")
		require.Len(t, paths, 1)
		original := strings.Replace(paths[0], "captures/", "originals/", 1)
		assert.Equal(t, capture, readFile(t, storage, original))
	})

	t.Run("keeps no separate original without an overlay", func(t *testing.T) {
		t.Parallel()

		cfg := captureConfig(1)
		cfg.SaveOriginals = true
		svc, storage := newTestCaptureService(t, cfg, "")
		run := startRun(t, svc)
		capture := testJPEG(t, 8, 8, color.White)

		_, err := svc.Create(t.Context(), run, &capture)

		require.NoError(t, err)
		svc.Wait()
		assert.Empty(t, listFiles(t, storage, "originals/"), "without an overlay, the stored capture already is the original")
	})

	t.Run("stores the capture without overlay when the overlay file is missing", func(t *testing.T) {
		t.Parallel()

		cfg := captureConfig(1)
		cfg.SaveOriginals = true
		svc, storage := newTestCaptureService(t, cfg, "overlays/missing.png")
		run := startRun(t, svc)
		capture := testJPEG(t, 8, 8, color.White)

		_, err := svc.Create(t.Context(), run, &capture)

		require.NoError(t, err)
		svc.Wait()
		paths := listFiles(t, storage, "captures/")
		require.Len(t, paths, 1)
		assert.Equal(t, capture, readFile(t, storage, paths[0]))
		assert.Empty(t, listFiles(t, storage, "originals/"), "without an applied overlay, there is no separate original")
	})

	t.Run("accepts the next run after a failed store", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(2), "overlays/frame.png")
		putFile(t, storage, "overlays/frame.png", testPNG(t, image.NewRGBA(image.Rect(0, 0, 2, 2))))
		owner := apitest.Faker(t).UUID()
		fake := []byte{0xFF, 0xD8, 0x01, 0x02, 0xFF, 0xD9}
		capture := testJPEG(t, 8, 8, color.White)
		first, err := svc.CreateRun(t.Context(), owner)
		require.NoError(t, err)
		_, err = svc.Create(t.Context(), first.Run, &fake)
		require.NoError(t, err)
		svc.Wait()
		next, err := svc.CreateRun(t.Context(), owner)
		require.NoError(t, err)

		_, err = svc.Create(t.Context(), next.Run, &capture)

		assert.NoError(t, err, "a failed store does not carry over to the next run")
	})

	t.Run("rejects a truncated capture without storing it", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(2), "")
		run := startRun(t, svc)
		truncated := testJPEG(t, 8, 8, color.White)
		truncated = truncated[:len(truncated)-10]

		_, err := svc.Create(t.Context(), run, &truncated)

		assert.ErrorIs(t, err, ErrCaptureCorrupt)
		svc.Wait()
		assert.Empty(t, listFiles(t, storage, "captures/"))
	})

	t.Run("keeps the image index of a rejected capture", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(2), "")
		run := startRun(t, svc)
		truncated := testJPEG(t, 8, 8, color.White)
		truncated = truncated[:len(truncated)-10]
		_, err := svc.Create(t.Context(), run, &truncated)
		require.ErrorIs(t, err, ErrCaptureCorrupt)
		valid := testJPEG(t, 8, 8, color.White)

		saved, err := svc.Create(t.Context(), run, &valid)

		require.NoError(t, err)
		assert.Equal(t, 1, saved.ImageIndex, "a rejected capture does not use up an image index")
	})

	t.Run("rejects a run that was never started", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(1), "")
		jpegData := testJPEG(t, 8, 8, color.White)

		_, err := svc.Create(t.Context(), 1, &jpegData)

		assert.ErrorIs(t, err, ErrRunNotOpen)
	})

	t.Run("rejects the zero run while no run is open", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(1), "")
		jpegData := testJPEG(t, 8, 8, color.White)

		_, err := svc.Create(t.Context(), 0, &jpegData)

		assert.ErrorIs(t, err, ErrRunNotOpen, "no run is open, so the zero run is no open run either")
	})

	t.Run("rejects a capture after the last image of the run", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(1), "")
		jpegData := testJPEG(t, 8, 8, color.White)
		run := startRun(t, svc)
		_, err := svc.Create(t.Context(), run, &jpegData)
		require.NoError(t, err)

		_, err = svc.Create(t.Context(), run, &jpegData)

		assert.ErrorIs(t, err, ErrRunNotOpen)
	})

	t.Run("rejects the next capture after one that only looks like a JPEG", func(t *testing.T) {
		t.Parallel()

		svc, storage, run := withOverlay(t, false)
		fake := []byte{0xFF, 0xD8, 0x01, 0x02, 0xFF, 0xD9}
		capture := testJPEG(t, 80, 40, color.White)
		_, err := svc.Create(t.Context(), run, &fake)
		require.NoError(t, err, "the capture is checked further while it is stored in the background")

		_, err = svc.Create(t.Context(), run, &capture)

		assert.ErrorIs(t, err, ErrCaptureNotStored, "the run does not go on with a photo missing")
		svc.Wait()
		assert.Empty(t, listFiles(t, storage, "captures/"))
	})
}

func TestLooksLikeCompleteJPEG(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "accepts start and end markers",
			data: []byte{0xFF, 0xD8, 0x00, 0xFF, 0xD9},
			want: true,
		},
		{
			name: "rejects truncated data",
			data: []byte{0xFF, 0xD8, 0x00, 0x00},
			want: false,
		},
		{
			name: "rejects a missing start",
			data: []byte{0x00, 0xD8, 0xFF, 0xD9},
			want: false,
		},
		{
			name: "rejects data that is too short",
			data: []byte{0xFF, 0xD8, 0xD9},
			want: false,
		},
		{
			name: "rejects empty data",
			data: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			complete := looksLikeCompleteJPEG(tt.data)

			assert.Equal(t, tt.want, complete)
		})
	}
}

func TestExifSegments(t *testing.T) {
	t.Parallel()

	t.Run("returns nothing for a JPEG without metadata", func(t *testing.T) {
		t.Parallel()

		plain := testJPEG(t, 8, 8, color.White)

		segments := exifSegments(plain)

		assert.Nil(t, segments, "an encoded JPEG carries no metadata")
	})

	t.Run("ignores a segment longer than the data", func(t *testing.T) {
		t.Parallel()

		segments := exifSegments([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0xFF, 0xFF})

		assert.Nil(t, segments)
	})
}

func TestWithoutThumbnail(t *testing.T) {
	t.Parallel()

	t.Run("removes the link to the thumbnail", func(t *testing.T) {
		t.Parallel()

		// A little endian TIFF structure whose first directory has one entry and links to a second one
		tiff := []byte{
			'I', 'I', 42, 0, 8, 0, 0, 0,
			1, 0,
			0x0f, 0x01, 2, 0, 4, 0, 0, 0, 'P', 'i', 0, 0,
			26, 0, 0, 0,
		}
		payload := append(append([]byte{}, exifHeader...), tiff...)
		segment := append([]byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}, payload...)

		result := withoutThumbnail(segment)

		link := 4 + len(exifHeader) + 8 + 2 + 12
		assert.Equal(t, []byte{0, 0, 0, 0}, result[link:link+4], "the first directory no longer links to the thumbnail")
		assert.Equal(t, []byte{26, 0, 0, 0}, segment[link:link+4], "the source stays unchanged")
		assert.Equal(t, segment[:link], result[:link], "everything before the link stays the same")
	})

	t.Run("keeps a segment that cannot be read", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			tiff []byte
		}{
			{
				name: "without a header",
				tiff: nil,
			},
			{
				name: "of an unknown byte order",
				tiff: []byte{
					'X', 'X', 42, 0, 8, 0, 0, 0,
					0, 0,
					0, 0, 0, 0,
				},
			},
			{
				name: "with a directory beyond the segment",
				tiff: []byte{'I', 'I', 42, 0, 64, 0, 0, 0},
			},
			{
				name: "with a link beyond the segment",
				tiff: []byte{
					'I', 'I', 42, 0, 8, 0, 0, 0,
					1, 0,
					0x0f, 0x01, 2, 0, 4, 0, 0, 0, 'P', 'i', 0, 0,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				payload := append(append([]byte{}, exifHeader...), tt.tiff...)
				broken := append([]byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}, payload...)

				result := withoutThumbnail(broken)

				assert.Equal(t, broken, result)
			})
		}
	})
}

func TestWithExif(t *testing.T) {
	t.Parallel()

	t.Run("keeps a JPEG without segments unchanged", func(t *testing.T) {
		t.Parallel()

		plain := testJPEG(t, 8, 8, color.White)

		result := withExif(plain, nil)

		assert.Equal(t, plain, result)
	})
}

func TestCaptureService_CreateRun(t *testing.T) {
	t.Parallel()

	t.Run("returns the first run", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(3), "")

		start, err := svc.CreateRun(t.Context(), apitest.Faker(t).UUID())

		require.NoError(t, err)
		assert.Equal(t, RunStart{
			Run:          1,
			ImagesPerRun: 3,
		}, start)
	})

	t.Run("continues after the highest stored run of any event", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(3), "")
		putFile(t, storage, "captures/earlier-event/x_000041-01_20260101-000000.jpg", []byte("x"))

		start, err := svc.CreateRun(t.Context(), apitest.Faker(t).UUID())

		require.NoError(t, err)
		assert.Equal(t, 42, start.Run)
	})

	t.Run("uses up the number of a run without captures", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(3), "")
		owner := apitest.Faker(t).UUID()
		first, err := svc.CreateRun(t.Context(), owner)
		require.NoError(t, err)

		second, err := svc.CreateRun(t.Context(), owner)

		require.NoError(t, err)
		assert.Equal(t, first.Run+1, second.Run)
	})

	t.Run("discards the incomplete run of the same client", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(3), "")
		owner := apitest.Faker(t).UUID()
		jpegData := testJPEG(t, 8, 8, color.White)
		first, err := svc.CreateRun(t.Context(), owner)
		require.NoError(t, err)
		_, err = svc.Create(t.Context(), first.Run, &jpegData)
		require.NoError(t, err)

		second, err := svc.CreateRun(t.Context(), owner)

		require.NoError(t, err)
		assert.Equal(t, first.Run+1, second.Run)
		assert.Empty(t, listFiles(t, storage, "captures/"), "the incomplete run is deleted")
	})

	t.Run("closes the incomplete run of the same client", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(3), "")
		owner := apitest.Faker(t).UUID()
		jpegData := testJPEG(t, 8, 8, color.White)
		first, err := svc.CreateRun(t.Context(), owner)
		require.NoError(t, err)
		_, err = svc.Create(t.Context(), first.Run, &jpegData)
		require.NoError(t, err)
		_, err = svc.CreateRun(t.Context(), owner)
		require.NoError(t, err)

		_, err = svc.Create(t.Context(), first.Run, &jpegData)

		assert.ErrorIs(t, err, ErrRunNotOpen)
	})

	t.Run("keeps a completed run when the next one starts", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(1), "")
		owner := apitest.Faker(t).UUID()
		jpegData := testJPEG(t, 8, 8, color.White)
		first, err := svc.CreateRun(t.Context(), owner)
		require.NoError(t, err)
		_, err = svc.Create(t.Context(), first.Run, &jpegData)
		require.NoError(t, err)

		_, err = svc.CreateRun(t.Context(), owner)

		require.NoError(t, err)
		assert.Len(t, listFiles(t, storage, "captures/"), 1)
	})

	t.Run("starts after an idle run of another client", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(3), "")
		f := apitest.Faker(t)
		first, err := svc.CreateRun(t.Context(), f.UUID())
		require.NoError(t, err)
		svc.lastActivity = time.Now().Add(-runIdleTimeout - time.Second)

		second, err := svc.CreateRun(t.Context(), f.UUID())

		require.NoError(t, err)
		assert.Equal(t, first.Run+1, second.Run)
	})

	t.Run("rejects a start while another client has an open run", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(3), "")
		f := apitest.Faker(t)
		_, err := svc.CreateRun(t.Context(), f.UUID())
		require.NoError(t, err)

		_, err = svc.CreateRun(t.Context(), f.UUID())

		assert.ErrorIs(t, err, ErrRunBusy)
	})
}

func TestCaptureService_AbortRun(t *testing.T) {
	t.Parallel()

	t.Run("removes the run", func(t *testing.T) {
		t.Parallel()

		cfg := captureConfig(3)
		cfg.SaveOriginals = true
		svc, storage := newTestCaptureService(t, cfg, "overlays/frame.png")
		putFile(t, storage, "overlays/frame.png", testPNG(t, image.NewRGBA(image.Rect(0, 0, 2, 2))))
		kept := "captures/summer-party/kept_000001-01_20260101-000000.jpg"
		putFile(t, storage, kept, []byte("x"))
		// Run 1 is stored already, so the aborted run is run 2
		run := startRun(t, svc)
		capture := testJPEG(t, 8, 8, color.White)
		_, err := svc.Create(t.Context(), run, &capture)
		require.NoError(t, err)
		putFile(t, storage, "prints/summer-party/booth_"+pad(run)+"-collage-1_0a1b2c3d.jpg", []byte("x"))

		err = svc.AbortRun(t.Context(), run)

		require.NoError(t, err)
		assert.Equal(t, []string{kept}, listFiles(t, storage, "captures/"))
		assert.Empty(t, listFiles(t, storage, "originals/"))
		assert.Empty(t, listFiles(t, storage, "prints/"), "the collages of the run are deleted as well")
	})

	t.Run("waits for the capture being stored", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(2), "")
		run := startRun(t, svc)
		capture := testJPEG(t, 8, 8, color.White)
		_, err := svc.Create(t.Context(), run, &capture)
		require.NoError(t, err)

		err = svc.AbortRun(t.Context(), run)

		require.NoError(t, err)
		svc.Wait()
		assert.Empty(t, listFiles(t, storage, "captures/"), "no capture is written after the run was deleted")
	})

	t.Run("closes the run", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(3), "")
		run := startRun(t, svc)
		capture := testJPEG(t, 8, 8, color.White)
		_, err := svc.Create(t.Context(), run, &capture)
		require.NoError(t, err)
		require.NoError(t, svc.AbortRun(t.Context(), run))

		_, err = svc.Create(t.Context(), run, &capture)

		assert.ErrorIs(t, err, ErrRunNotOpen, "an aborted run accepts no further captures")
	})

	t.Run("ignores an unknown run", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestCaptureService(t, captureConfig(3), "")

		err := svc.AbortRun(t.Context(), 99)

		require.NoError(t, err, "aborting an unknown run is not an error")
	})
}

func TestCaptureService_PrepareOverlay(t *testing.T) {
	t.Parallel()

	t.Run("scales the overlay to the fixed capture size ahead of the first capture", func(t *testing.T) {
		t.Parallel()

		cfg := captureConfig(1)
		cfg.CaptureWidth = 4000
		cfg.CaptureHeight = 2400
		svc, storage := newTestCaptureService(t, cfg, "overlays/frame.png")
		putFile(t, storage, "overlays/frame.png", testPNG(t, image.NewRGBA(image.Rect(0, 0, 4, 2))))

		svc.PrepareOverlay(t.Context())

		require.NotNil(t, svc.overlay)
		assert.Contains(t, svc.overlay.scaled, image.Rect(0, 0, 4000, 2400), "the size of the captures")
		assert.Contains(t, svc.overlay.scaled, image.Rect(0, 0, 800, 480), "the size of their screen versions")
	})

	t.Run("prepares nothing without a fixed capture size", func(t *testing.T) {
		t.Parallel()

		svc, storage := newTestCaptureService(t, captureConfig(1), "overlays/frame.png")
		putFile(t, storage, "overlays/frame.png", testPNG(t, image.NewRGBA(image.Rect(0, 0, 4, 2))))

		svc.PrepareOverlay(t.Context())

		assert.Nil(t, svc.overlay)
	})

	t.Run("prepares nothing for a missing overlay", func(t *testing.T) {
		t.Parallel()

		cfg := captureConfig(1)
		cfg.CaptureWidth = 40
		cfg.CaptureHeight = 20
		svc, _ := newTestCaptureService(t, cfg, "overlays/missing.png")

		svc.PrepareOverlay(t.Context())

		assert.Nil(t, svc.overlay, "a missing overlay is no failure")
	})
}

func TestPreviewBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		bounds image.Rectangle
		want   image.Rectangle
	}{
		{
			name:   "scales a wide image to the screen",
			bounds: image.Rect(0, 0, 4000, 2400),
			want:   image.Rect(0, 0, 800, 480),
		},
		{
			name:   "scales a narrower image to the screen height",
			bounds: image.Rect(0, 0, 4056, 3040),
			want:   image.Rect(0, 0, 640, 480),
		},
		{
			name:   "keeps the size of a small image",
			bounds: image.Rect(0, 0, 320, 200),
			want:   image.Rect(0, 0, 320, 200),
		},
		{
			name:   "returns one pixel for an empty image",
			bounds: image.Rect(0, 0, 0, 200),
			want:   image.Rect(0, 0, 1, 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bounds := previewBounds(tt.bounds)

			assert.Equal(t, tt.want, bounds)
		})
	}
}

// newTestCaptureService builds a capture service on a fresh local storage.
func newTestCaptureService(t *testing.T, cfg CaptureOptions, overlayPath string) (*CaptureService, media.Storage) {
	t.Helper()

	storage := newTestStorage(t)
	cfg.OverlayPath = overlayPath
	svc := newCaptureServiceOn(storage, cfg)
	// The cleanups run in reverse order, so the background stores end before the storage is removed
	t.Cleanup(svc.Wait)
	return svc, storage
}

// startRun opens a run of a new client and fails the test on an error.
func startRun(t *testing.T, svc *CaptureService) int {
	t.Helper()

	start, err := svc.CreateRun(t.Context(), apitest.Faker(t).UUID())
	require.NoError(t, err)
	return start.Run
}
