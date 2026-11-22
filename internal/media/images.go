package media

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // The PNG decoder registers itself for image.Decode, so WithFormats can accept PNG
	"io"
	"slices"

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
)

const (
	// defaultMaxImagePixels caps the decoded pixel count unless the caller sets its own budget with WithMaxPixels.
	// Decoding allocates several bytes per pixel, so this bound protects the process from decompression-bomb images
	// that are small on disk but enormous once expanded.
	defaultMaxImagePixels = 50_000_000 // 50MP
)

var (
	// ErrUnsupportedFormat is returned when the data is no image in an accepted format.
	ErrUnsupportedFormat = errors.New("media: unsupported image format")
	// ErrImageTooLarge is returned when the uploaded image exceeds the maximum pixel budget.
	ErrImageTooLarge = errors.New("media: image resolution too high: exceeds the maximum pixel budget")
	// ErrImageTooSmall is returned when an edge of the uploaded image is shorter than the minimum dimension.
	ErrImageTooSmall = errors.New("media: image resolution too low: below the minimum dimension")
)

// ImageFormat is the name under which the image package registers a format.
type ImageFormat string

const (
	ImageFormatJPEG ImageFormat = "jpeg"
	ImageFormatPNG  ImageFormat = "png"
	ImageFormatWebP ImageFormat = "webp"
)

// DecodeOption configures the formats and resolution bounds of ValidateImageData and DecodeImage.
type DecodeOption func(*decodeOptions)

// decodeOptions holds the settings the DecodeOption values apply.
type decodeOptions struct {
	formats      []ImageFormat
	maxPixels    int64
	minDimension int
}

// WithFormats sets the formats an image may have. Without it, JPEG and WebP are accepted.
func WithFormats(formats ...ImageFormat) DecodeOption {
	return func(o *decodeOptions) {
		o.formats = formats
	}
}

// WithMaxPixels sets the largest pixel count, width times height, an image may have.
// The memory a decode needs grows with the pixel count,
// so a caller that knows the size of its uploads sets a budget just above it.
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

// ValidateImageData verifies that the data is an image in an accepted format within the resolution bounds, reading only its header.
func ValidateImageData(data []byte, opts ...DecodeOption) error {
	_, err := decodeHeader(bytes.NewReader(data), newDecodeOptions(opts))
	return err
}

// DecodeImage decodes an image in an accepted format from r once for every later use, and fails with ErrImageTooLarge
// or ErrImageTooSmall before any pixel memory is allocated, or ErrUnsupportedFormat.
func DecodeImage(r io.Reader, opts ...DecodeOption) (image.Image, error) {
	// The header is read through a copy, so the full decode can replay the bytes the header check consumed
	var header bytes.Buffer
	if _, err := decodeHeader(io.TeeReader(r, &header), newDecodeOptions(opts)); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(io.MultiReader(&header, r))
	if err != nil {
		return nil, ErrUnsupportedFormat
	}
	return img, nil
}

// decodeHeader reads the dimensions and the format of an image from r and checks them against the options.
func decodeHeader(r io.Reader, o decodeOptions) (image.Config, error) {
	cfg, format, err := image.DecodeConfig(r)
	if err != nil || !slices.Contains(o.formats, ImageFormat(format)) {
		return image.Config{}, ErrUnsupportedFormat
	}
	return cfg, checkResolution(cfg, o)
}

// newDecodeOptions applies the options to the defaults.
func newDecodeOptions(opts []DecodeOption) decodeOptions {
	o := decodeOptions{
		formats: []ImageFormat{
			ImageFormatJPEG,
			ImageFormatWebP,
		},
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

// ScaleDown downscales img with a Catmull-Rom filter so that neither edge exceeds maxEdge, keeping the aspect ratio.
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

// ScaleDownSteps downscales img to each of the edges, given from the largest to the smallest, like ScaleDown.
// Every step derives from the previous one, so only the first step reads the full image,
// and the caller may drop the full image as soon as the steps exist.
func ScaleDownSteps(img image.Image, edges ...int) []image.Image {
	steps := make([]image.Image, 0, len(edges))
	for _, edge := range edges {
		img = ScaleDown(img, edge)
		steps = append(steps, img)
	}
	return steps
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

// EncodeWebP encodes an image as a lossy WebP at the given quality, which also strips all metadata.
func EncodeWebP(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	// The encoder treats an unset method as the fastest one, so the default is set explicitly
	if err := webp.Encode(&buf, img, webp.Options{
		Quality: quality,
		Method:  webp.DefaultMethod,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// dHashWidth and dHashHeight are the sample dimensions for the difference hash.
// Comparing each pixel to its right neighbor across a 9x8 grayscale grid yields exactly 8x8 = 64 comparison bits.
const (
	dHashWidth  = 9
	dHashHeight = 8
)

// PerceptualHash computes a 64-bit difference hash of a decoded image, which is a few bits apart for visually similar images.
func PerceptualHash(img image.Image) uint64 {
	// The image is downscaled to a small grayscale grid, and one bit is set per horizontal brightness gradient,
	// so comparing two hashes is a Hamming-distance check.
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
