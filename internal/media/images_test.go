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

	t.Run("accepts a WebP", func(t *testing.T) {
		t.Parallel()

		data := makeWebP(t, 512, 512)

		err := ValidateImageData(data)

		require.NoError(t, err)
	})

	t.Run("accepts a format the caller allows", func(t *testing.T) {
		t.Parallel()

		data := makePNG(t, 512, 512)

		err := ValidateImageData(data, WithFormats(ImageFormatPNG))

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

		data := makePNG(t, 512, 512)

		err := ValidateImageData(data)

		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("rejects a format the caller does not allow", func(t *testing.T) {
		t.Parallel()

		data := makeWebP(t, 512, 512)

		err := ValidateImageData(data, WithFormats(ImageFormatJPEG))

		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("rejects data that is no image", func(t *testing.T) {
		t.Parallel()

		err := ValidateImageData([]byte("this is not an image"))

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

	t.Run("returns a JPEG", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		img, err := DecodeImage(bytes.NewReader(data))

		require.NoError(t, err)
		assert.Equal(t, image.Pt(600, 512), img.Bounds().Size())
	})

	t.Run("returns a WebP", func(t *testing.T) {
		t.Parallel()

		data := makeWebP(t, 600, 512)

		img, err := DecodeImage(bytes.NewReader(data))

		require.NoError(t, err)
		assert.Equal(t, image.Pt(600, 512), img.Bounds().Size())
	})

	t.Run("rejects a format the caller does not allow", func(t *testing.T) {
		t.Parallel()

		data := makeWebP(t, 600, 512)

		_, err := DecodeImage(bytes.NewReader(data), WithFormats(ImageFormatJPEG))

		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("rejects an image beyond the pixel budget of the caller", func(t *testing.T) {
		t.Parallel()

		data := makeJPEG(t, 600, 512)

		_, err := DecodeImage(bytes.NewReader(data), WithMaxPixels(600*512-1))

		assert.ErrorIs(t, err, ErrImageTooLarge)
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

func TestScaleDownSteps(t *testing.T) {
	t.Parallel()

	t.Run("bounds every step by its edge", func(t *testing.T) {
		t.Parallel()

		img := image.NewRGBA(image.Rect(0, 0, 2000, 1200))

		steps := ScaleDownSteps(img, 1600, 400)

		require.Len(t, steps, 2)
		assert.Equal(t, image.Pt(1600, 960), steps[0].Bounds().Size())
		assert.Equal(t, image.Pt(400, 240), steps[1].Bounds().Size())
	})

	t.Run("keeps a step the image already fits unscaled", func(t *testing.T) {
		t.Parallel()

		img := image.NewRGBA(image.Rect(0, 0, 800, 600))

		steps := ScaleDownSteps(img, 1600, 400)

		require.Len(t, steps, 2)
		assert.Same(t, img, steps[0], "not upscaled")
		assert.Equal(t, image.Pt(400, 300), steps[1].Bounds().Size())
	})

	t.Run("returns no steps without edges", func(t *testing.T) {
		t.Parallel()

		img := image.NewRGBA(image.Rect(0, 0, 800, 600))

		steps := ScaleDownSteps(img)

		assert.Empty(t, steps)
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

func TestEncodeWebP(t *testing.T) {
	t.Parallel()

	t.Run("encodes a decodable webp of the same size", func(t *testing.T) {
		t.Parallel()

		img := image.NewRGBA(image.Rect(0, 0, 640, 480))

		data, err := EncodeWebP(img, 85)

		require.NoError(t, err)
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, "webp", format)
		assert.Equal(t, [2]int{640, 480}, [2]int{cfg.Width, cfg.Height})
	})

	t.Run("writes a simple lossy webp without metadata chunks", func(t *testing.T) {
		t.Parallel()

		img := decodeImageOrFail(t, makeJPEG(t, 640, 480))

		data, err := EncodeWebP(img, 85)

		require.NoError(t, err)
		// Only the extended format behind a VP8X chunk can carry EXIF or XMP chunks
		require.Greater(t, len(data), 16)
		assert.Equal(t, "VP8 ", string(data[12:16]))
	})

	t.Run("spends more bytes at a higher quality", func(t *testing.T) {
		t.Parallel()

		img := decodeImageOrFail(t, makePatternJPEG(t, 640, 480, 95, wavyPattern))
		low, err := EncodeWebP(img, 50)
		require.NoError(t, err)

		high, err := EncodeWebP(img, 90)

		require.NoError(t, err)
		assert.Greater(t, len(high), len(low))
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

	t.Run("keeps a webp re-encoding similar to its source", func(t *testing.T) {
		t.Parallel()

		source := decodeImageOrFail(t, makePatternJPEG(t, 512, 512, 95, wavyPattern))
		encoded, err := EncodeWebP(ScaleDown(source, 400), 85)
		require.NoError(t, err)
		h1 := PerceptualHash(source)

		h2 := PerceptualHash(decodeImageOrFail(t, encoded))

		assert.LessOrEqual(t, hammingDistance(h1, h2), maxSimilarDistance, "the stored webp stays similar to the upload")
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
// Using normalized coordinates means the same pattern rendered at different sizes is the *same image* at a different resolution,
// which is what the perceptual hash must treat as similar.
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

// wavyPattern is a smooth, low-frequency brightness field. Its gentle structure survives downscaling to the hash grid
// (unlike a high-frequency pattern, which aliases differently per resolution) while still producing mixed gradient bits.
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

// makeWebP encodes a blank WebP of the given dimensions for use as test input.
func makeWebP(t *testing.T, width, height int) []byte {
	t.Helper()

	data, err := EncodeWebP(image.NewRGBA(image.Rect(0, 0, width, height)), 80)
	require.NoError(t, err)
	return data
}

// makePNG encodes a blank PNG of the given dimensions for use as test input.
func makePNG(t *testing.T, width, height int) []byte {
	t.Helper()

	buf := &bytes.Buffer{}
	require.NoError(t, png.Encode(buf, image.NewRGBA(image.Rect(0, 0, width, height))))
	return buf.Bytes()
}
