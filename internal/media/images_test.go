package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"math/bits"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateImageData(t *testing.T) {
	t.Parallel()

	t.Run("accepts a JPEG", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 512, 512)

		err := ValidateImageData(data)

		require.NoError(t, err)
	})

	t.Run("accepts an image within the pixel budget of the caller", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		err := ValidateImageData(data, WithMaxPixels(600*512))

		require.NoError(t, err)
	})

	t.Run("accepts an image at the minimum dimension of the caller", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		err := ValidateImageData(data, WithMinDimension(512))

		require.NoError(t, err)
	})

	t.Run("rejects a PNG", func(t *testing.T) {
		t.Parallel()

		img := image.NewNRGBA(image.Rect(0, 0, 512, 512))
		for y := range 512 {
			for x := range 512 {
				img.SetNRGBA(x, y, color.NRGBA{
					R: 100,
					G: 149,
					B: 237,
					A: 255,
				})
			}
		}
		buf := &bytes.Buffer{}
		require.NoError(t, png.Encode(buf, img))

		err := ValidateImageData(buf.Bytes())

		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("rejects an image below the minimum dimension of the caller", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		err := ValidateImageData(data, WithMinDimension(513))

		assert.ErrorIs(t, err, ErrImageTooSmall)
	})

	t.Run("rejects an image beyond the pixel budget of the caller", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		err := ValidateImageData(data, WithMaxPixels(600*512-1))

		assert.ErrorIs(t, err, ErrImageTooLarge)
	})
}

func TestDecodeImage(t *testing.T) {
	t.Parallel()

	t.Run("returns the image within the budget", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		img, err := DecodeImage(bytes.NewReader(data), WithMaxPixels(600*512))

		require.NoError(t, err)
		assert.Equal(t, 600, img.Bounds().Dx())
		assert.Equal(t, 512, img.Bounds().Dy())
	})

	t.Run("rejects a format other than JPEG", func(t *testing.T) {
		t.Parallel()

		_, err := DecodeImage(bytes.NewReader([]byte("this is not an image")))

		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("rejects an image beyond the budget", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		_, err := DecodeImage(bytes.NewReader(data), WithMaxPixels(600*512-1))

		assert.ErrorIs(t, err, ErrImageTooLarge)
	})

	t.Run("rejects an image below the minimum dimension", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		_, err := DecodeImage(bytes.NewReader(data), WithMinDimension(513))

		assert.ErrorIs(t, err, ErrImageTooSmall)
	})
}

func TestScaleDown(t *testing.T) {
	t.Parallel()

	t.Run("bounds the longer edge of a landscape image", func(t *testing.T) {
		t.Parallel()

		img := image.NewRGBA(image.Rect(0, 0, 2000, 1200))

		scaled := ScaleDown(img, 1600)

		assert.Equal(t, image.Pt(1600, 960), scaled.Bounds().Size())
	})

	t.Run("bounds the longer edge of a portrait image", func(t *testing.T) {
		t.Parallel()

		img := image.NewRGBA(image.Rect(0, 0, 1200, 2000))

		scaled := ScaleDown(img, 400)

		assert.Equal(t, image.Pt(240, 400), scaled.Bounds().Size())
	})

	t.Run("keeps a small image unscaled", func(t *testing.T) {
		t.Parallel()

		img := image.NewRGBA(image.Rect(0, 0, 800, 600))

		scaled := ScaleDown(img, 1600)

		assert.Same(t, img, scaled, "not upscaled")
	})
}

func TestEncodeJPEG(t *testing.T) {
	t.Parallel()

	t.Run("encodes a decodable jpeg of the same size", func(t *testing.T) {
		t.Parallel()

		img := image.NewRGBA(image.Rect(0, 0, 640, 480))

		data, err := EncodeJPEG(img, 82)

		require.NoError(t, err)
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, "jpeg", format)
		assert.Equal(t, [2]int{640, 480}, [2]int{cfg.Width, cfg.Height})
	})
}

// maxSimilarDistance is the largest Hamming distance the tests accept between the hashes of the same picture.
const maxSimilarDistance = 8

func TestPerceptualHash(t *testing.T) {
	t.Parallel()

	t.Run("keeps re-encoded and rescaled variants similar", func(t *testing.T) {
		t.Parallel()

		highQuality := decodeImageOrFail(t, makePatternJPEG(t, 512, 512, 95, wavyPattern))
		lowQualityScaled := decodeImageOrFail(t, makePatternJPEG(t, 256, 200, 35, wavyPattern))

		h1 := PerceptualHash(highQuality)
		h2 := PerceptualHash(lowQualityScaled)

		assert.LessOrEqual(t, hammingDistance(h1, h2), maxSimilarDistance, "re-encoded and rescaled variants stay similar")
	})

	t.Run("tells different images apart", func(t *testing.T) {
		t.Parallel()

		// A transposed wave swaps which axis carries the higher frequency,
		// producing a genuinely different horizontal-gradient structure.
		transposed := func(fx, fy float64) uint8 {
			v := (math.Sin(fy*4*math.Pi) + math.Cos(fx*3*math.Pi) + 2) / 4 * 255
			return uint8(v)
		}
		original := decodeImageOrFail(t, makePatternJPEG(t, 256, 256, 90, wavyPattern))
		different := decodeImageOrFail(t, makePatternJPEG(t, 256, 256, 90, transposed))

		h1 := PerceptualHash(original)
		h2 := PerceptualHash(different)

		assert.Greater(t, hammingDistance(h1, h2), maxSimilarDistance, "different images are told apart")
	})
}

// decodeImageOrFail decodes encoded image bytes into an image or fails the test.
func decodeImageOrFail(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := DecodeImage(bytes.NewReader(data))
	require.NoError(t, err)
	return img
}

// makePatternJPEG encodes a grayscale JPEG whose brightness follows the given normalized-coordinate function.
// Using normalized coordinates means the same pattern rendered at different sizes is the *same image*
// at a different resolution, which is what the perceptual hash must treat as similar.
func makePatternJPEG(t *testing.T, width, height, quality int, brightness func(fx, fy float64) uint8) []byte {
	t.Helper()

	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			fx := float64(x) / float64(width-1)
			fy := float64(y) / float64(height-1)
			img.SetGray(x, y, color.Gray{
				Y: brightness(fx, fy),
			})
		}
	}

	buf := &bytes.Buffer{}
	require.NoError(t, jpeg.Encode(buf, img, &jpeg.Options{
		Quality: quality,
	}))
	return buf.Bytes()
}

// wavyPattern is a smooth, low-frequency brightness field. Its gentle structure survives downscaling
// to the hash grid (unlike a high-frequency pattern, which aliases differently per resolution)
// while still producing mixed gradient bits.
func wavyPattern(fx, fy float64) uint8 {
	v := (math.Sin(fx*4*math.Pi) + math.Cos(fy*3*math.Pi) + 2) / 4 * 255
	return uint8(v)
}

// hammingDistance counts the differing bits between two perceptual hashes.
func hammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// makeJPEG encodes a filled JPEG of the given dimensions for use as test input.
func makeJPEG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: 120,
				A: 255,
			})
		}
	}

	buf := &bytes.Buffer{}
	require.NoError(t, jpeg.Encode(buf, img, nil))
	return buf.Bytes()
}
