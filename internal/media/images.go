package media

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strings"

	xdraw "golang.org/x/image/draw"
)

const (
	// defaultMaxImagePixels caps the decoded pixel count unless the caller sets its own budget with WithMaxPixels.
	// Decoding allocates several bytes per pixel, so this bound protects the process from decompression-bomb images
	// that are small on disk but enormous once expanded.
	defaultMaxImagePixels = 50_000_000 // 50MP
)

var (
	// ErrUnsupportedFormat is returned when the uploaded file is not a JPEG image.
	ErrUnsupportedFormat = errors.New("media: unsupported image format: only JPEG is accepted")
	// ErrImageTooLarge is returned when the uploaded image exceeds the maximum pixel budget.
	ErrImageTooLarge = errors.New("media: image resolution too high: exceeds the maximum pixel budget")
	// ErrImageTooSmall is returned when an edge of the uploaded image is shorter than the minimum dimension.
	ErrImageTooSmall = errors.New("media: image resolution too low: below the minimum dimension")
)

// DecodeOption configures the resolution bounds of ValidateImageData and DecodeImage.
type DecodeOption func(*decodeOptions)

// decodeOptions holds the settings the DecodeOption values apply.
type decodeOptions struct {
	maxPixels    int64
	minDimension int
}

// WithMaxPixels sets the largest pixel count, width times height, an image may have. The memory a decode needs grows
// with the pixel count, so a caller that knows the size of its uploads sets a budget just above it.
func WithMaxPixels(maxPixels int64) DecodeOption {
	return func(o *decodeOptions) {
		o.maxPixels = maxPixels
	}
}

// WithMinDimension sets the shortest length both edges of an image must reach. Without it, any size is accepted.
func WithMinDimension(minDimension int) DecodeOption {
	return func(o *decodeOptions) {
		o.minDimension = minDimension
	}
}

// ValidateImageData verifies that the data is a JPEG within the accepted resolution bounds.
func ValidateImageData(data []byte, opts ...DecodeOption) error {
	if len(data) == 0 {
		return ErrUnsupportedFormat
	}
	ct := strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))
	if ct != "image/jpeg" {
		return ErrUnsupportedFormat
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return ErrUnsupportedFormat
	}

	return checkResolution(cfg, newDecodeOptions(opts))
}

// DecodeImage decodes a JPEG from r, mapping any decoding failure to ErrUnsupportedFormat.
// It reads the dimensions from the header first and returns ErrImageTooLarge or ErrImageTooSmall for an image
// outside the resolution bounds before any pixel memory is allocated, so every caller is protected,
// also one that skipped ValidateImageData. It decouples decoding from processing so an upload can be decoded once
// and the resulting image reused for both re-encoding and hashing.
func DecodeImage(r io.Reader, opts ...DecodeOption) (image.Image, error) {
	// The header is read through a copy, so the full decode can replay the bytes the header check consumed
	var header bytes.Buffer
	cfg, err := jpeg.DecodeConfig(io.TeeReader(r, &header))
	if err != nil {
		return nil, ErrUnsupportedFormat
	}
	if err := checkResolution(cfg, newDecodeOptions(opts)); err != nil {
		return nil, err
	}

	img, err := jpeg.Decode(io.MultiReader(&header, r))
	if err != nil {
		return nil, ErrUnsupportedFormat
	}
	return img, nil
}

// newDecodeOptions applies the options to the defaults.
func newDecodeOptions(opts []DecodeOption) decodeOptions {
	o := decodeOptions{
		maxPixels: defaultMaxImagePixels,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// checkResolution returns ErrImageTooLarge or ErrImageTooSmall for image dimensions outside the bounds of o.
func checkResolution(cfg image.Config, o decodeOptions) error {
	if int64(cfg.Width)*int64(cfg.Height) > o.maxPixels {
		return ErrImageTooLarge
	}
	if cfg.Width < o.minDimension || cfg.Height < o.minDimension {
		return ErrImageTooSmall
	}
	return nil
}

// ScaleDown downscales img so that neither edge exceeds maxEdge, preserving the aspect ratio
// with a high-quality Catmull-Rom filter. Images already within the bound are returned unchanged.
func ScaleDown(img image.Image, maxEdge int) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maxEdge && height <= maxEdge {
		return img
	}

	var newWidth, newHeight int
	if width >= height {
		newWidth = maxEdge
		newHeight = height * maxEdge / width
	} else {
		newHeight = maxEdge
		newWidth = width * maxEdge / height
	}
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Src, nil)
	return dst
}

// EncodeJPEG encodes an image as a baseline JPEG at the given quality, which also strips all metadata.
func EncodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{
		Quality: quality,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// dHashWidth and dHashHeight are the sample dimensions for the difference hash. Comparing each pixel
// to its right neighbor across a 9x8 grayscale grid yields exactly 8x8 = 64 comparison bits.
const (
	dHashWidth  = 9
	dHashHeight = 8
)

// PerceptualHash computes a 64-bit difference hash (dHash) of an already-decoded image.
// Unlike the cryptographic content hash, visually similar images (re-encoded, rescaled, or slightly adjusted)
// produce hashes only a few bits apart, so it finds copies of an image that a byte-exact hash would miss.
// The image is downscaled to a small grayscale grid, and one bit is set per horizontal brightness gradient.
// Comparing two hashes is then a Hamming-distance check.
func PerceptualHash(img image.Image) uint64 {
	gray := image.NewGray(image.Rect(0, 0, dHashWidth, dHashHeight))
	xdraw.CatmullRom.Scale(gray, gray.Bounds(), img, img.Bounds(), xdraw.Src, nil)

	var hash uint64
	bit := 0
	for y := range dHashHeight {
		for x := range dHashWidth - 1 {
			if gray.GrayAt(x, y).Y > gray.GrayAt(x+1, y).Y {
				hash |= uint64(1) << uint(63-bit)
			}
			bit++
		}
	}

	return hash
}
