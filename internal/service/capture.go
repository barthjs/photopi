package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"sync"
	"time"

	"github.com/rs/zerolog"
	xdraw "golang.org/x/image/draw"

	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/store"
)

const (
	// storeTimeout bounds compositing and storing one capture in the background.
	storeTimeout = time.Minute

	// previewWidth and previewHeight bound the screen version of a capture, sized for the kiosk screen.
	previewWidth  = 800
	previewHeight = 480
	// previewQuality is the JPEG quality of the screen version, which is only shown, never kept.
	previewQuality = 80
	// captureQuality is the JPEG quality of a stored capture with the overlay drawn in.
	captureQuality = 90

	// runIdleTimeout is how long an open run without a capture blocks another client from starting a run.
	// A guest needs at most a countdown per capture, so a longer pause means the client that opened the run is gone.
	runIdleTimeout = 2 * time.Minute
)

var (
	// ErrRunNotOpen is returned when a capture is submitted for a run that is not the currently open one,
	// for example, after it was already completed, aborted, or never started.
	ErrRunNotOpen = errors.New("service: run is not open for capture")
	// ErrRunBusy is returned when a client starts a run while another client's run is still open.
	ErrRunBusy = errors.New("service: another run is open")
	// ErrCaptureCorrupt is returned when a capture is not a well-formed JPEG,
	// for example, a still truncated by a flaky camera driver. It is never stored.
	ErrCaptureCorrupt = errors.New("service: capture is not a usable image")
	// ErrCaptureNotStored is returned for the capture after one that could not be stored,
	// so the run does not go on with a photo missing.
	ErrCaptureNotStored = errors.New("service: the previous capture could not be stored")
)

// jpegSOI and jpegEOI mark the start and end of a JPEG image.
var (
	jpegSOI = []byte{0xFF, 0xD8}
	jpegEOI = []byte{0xFF, 0xD9}
)

// CaptureOptions holds the settings of a CaptureService.
type CaptureOptions struct {
	// ImagesPerRun is the number of captures that complete a run.
	ImagesPerRun int
	// FilePrefix starts the filename of every stored capture.
	FilePrefix string
	// EventName is slugified into the storage folder captures are filed under.
	EventName string
	// SaveOriginals also stores the capture without the overlay, under "originals/".
	SaveOriginals bool
	// OverlayPath is the storage path of the PNG drawn over every capture, empty for none.
	OverlayPath string
	// CaptureWidth and CaptureHeight are the size of the captures if the camera has a fixed one, zero otherwise.
	// With a fixed size, PrepareOverlay scales the overlay before the first capture.
	CaptureWidth  int
	CaptureHeight int
}

// CaptureService allocates run numbers, takes the captures of the open run, and stores them.
type CaptureService struct {
	logger    zerolog.Logger
	options   CaptureOptions
	runs      *store.RunStore
	captures  *store.CaptureStore
	overlays  *store.OverlayStore
	eventSlug string

	// mu serializes the runs: starting, capturing, and aborting.
	mu           sync.Mutex
	openRun      int
	openOwner    string
	nextIndex    int
	lastActivity time.Time

	// stored is closed once the capture that is composited and stored in the background is written, nil while none is.
	// Every step that reads or changes the files of a run waits for it first.
	stored chan struct{}
	// storeErr is the failure of the last background store, set before stored is closed.
	storeErr error
	// storing tracks the background stores, so Wait can let them finish before the process ends.
	storing sync.WaitGroup

	// overlayMu guards overlay, which the background stores and PrepareOverlay use.
	overlayMu sync.Mutex
	overlay   *overlayCache

	// previewMu guards the screen versions of the captures of the current run,
	// which the kiosk shows right after a shot and on the review screen.
	// They live in memory only and are replaced by the next run.
	previewMu     sync.Mutex
	previewRun    int
	previews      map[int][]byte
	previewsReady map[int]chan struct{}
}

// NewCaptureService creates a CaptureService.
func NewCaptureService(
	logger zerolog.Logger,
	options CaptureOptions,
	runs *store.RunStore,
	captures *store.CaptureStore,
	overlays *store.OverlayStore,
) *CaptureService {
	return &CaptureService{
		logger:    logger,
		options:   options,
		runs:      runs,
		captures:  captures,
		overlays:  overlays,
		eventSlug: slugify(options.EventName),
	}
}

// ListRunFiles returns the storage paths of every capture stored for a run, ordered by image index.
func (s *CaptureService) ListRunFiles(ctx context.Context, run int) ([]string, error) {
	if err := s.awaitStore(ctx); err != nil {
		return nil, err
	}

	return s.captures.ListByRun(ctx, run, false)
}

// Get reads one stored capture of a run or returns apperr.ErrNotFound if it does not exist.
func (s *CaptureService) Get(ctx context.Context, run, imageIndex int) ([]byte, error) {
	if err := s.awaitStore(ctx); err != nil {
		return nil, err
	}

	return s.captures.Get(ctx, run, imageIndex, false)
}

// awaitStore waits until the capture stored in the background is written,
// for the steps that read the files of a run without changing the runs.
func (s *CaptureService) awaitStore(ctx context.Context) error {
	s.mu.Lock()
	stored := s.stored
	s.mu.Unlock()

	if stored == nil {
		return nil
	}

	select {
	case <-stored:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetPreview returns the screen version of one capture of a run, a JPEG with the overlay that fits the kiosk screen.
// It waits for a capture that is still being stored and renders a missing one from the stored capture.
func (s *CaptureService) GetPreview(ctx context.Context, run, imageIndex int) ([]byte, error) {
	if err := s.awaitPreview(ctx, run, imageIndex); err != nil {
		return nil, err
	}

	s.previewMu.Lock()
	cached := s.previews[imageIndex]
	if run != s.previewRun {
		cached = nil
	}
	s.previewMu.Unlock()
	if cached != nil {
		return cached, nil
	}

	stored, err := s.Get(ctx, run, imageIndex)
	if err != nil {
		return nil, err
	}

	base, err := jpeg.Decode(bytes.NewReader(stored))
	if err != nil {
		return nil, fmt.Errorf("service: decoding run %d image %d: %w", run, imageIndex, err)
	}

	// The stored capture carries the overlay already
	preview, err := s.renderPreview(ctx, base, false)
	if err != nil {
		return nil, err
	}
	s.publishPreview(run, imageIndex, preview)

	return preview, nil
}

// awaitPreview waits until the screen version of a capture that is still being stored exists.
func (s *CaptureService) awaitPreview(ctx context.Context, run, imageIndex int) error {
	s.previewMu.Lock()
	ready := s.previewsReady[imageIndex]
	if run != s.previewRun {
		ready = nil
	}
	s.previewMu.Unlock()

	if ready == nil {
		return nil
	}

	select {
	case <-ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Capture is a stored capture with its position in the run.
type Capture struct {
	Run          int
	ImageIndex   int
	ImagesPerRun int
}

// Create takes data as the next capture of the open run and returns ErrRunNotOpen for any other run.
// It returns once the capture is checked and numbered and stores it in the background,
// so the guests may move again as early as possible. A failed store fails the next Create with ErrCaptureNotStored.
func (s *CaptureService) Create(ctx context.Context, run int, data *[]byte) (Capture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if run == 0 || run != s.openRun {
		return Capture{}, ErrRunNotOpen
	}
	s.lastActivity = time.Now()

	if err := s.awaitStoreLocked(ctx); err != nil {
		return Capture{}, err
	}
	if s.storeErr != nil {
		s.storeErr = nil
		return Capture{}, ErrCaptureNotStored
	}

	if !looksLikeCompleteJPEG(*data) {
		return Capture{}, ErrCaptureCorrupt
	}

	imageIndex := s.nextIndex
	s.nextIndex++
	if s.nextIndex > s.options.ImagesPerRun {
		s.openRun = 0
	}
	previewReady := s.expectPreview(run, imageIndex)

	stored := make(chan struct{})
	s.stored = stored
	// The store outlives the request, which ends with the response
	storeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), storeTimeout)
	s.storing.Go(func() {
		defer cancel()
		defer close(stored)

		s.storeErr = s.store(storeCtx, run, imageIndex, data, previewReady)
		if s.storeErr != nil {
			s.logger.Error().Err(s.storeErr).Int("run", run).Int("image", imageIndex).Msg("storing capture failed")
		}
	})

	return Capture{
		Run:          run,
		ImageIndex:   imageIndex,
		ImagesPerRun: s.options.ImagesPerRun,
	}, nil
}

// looksLikeCompleteJPEG reports whether data starts and ends with the JPEG start and end markers.
// The check is cheap enough for every capture and catches a truncated still before it is stored.
func looksLikeCompleteJPEG(data []byte) bool {
	return len(data) >= 4 &&
		bytes.Equal(data[:2], jpegSOI) &&
		bytes.Equal(data[len(data)-2:], jpegEOI)
}

// expectPreview registers the screen version of a capture that is about to be stored
// and returns the channel the store closes once it exists.
func (s *CaptureService) expectPreview(run, imageIndex int) chan struct{} {
	s.previewMu.Lock()
	defer s.previewMu.Unlock()

	ready := make(chan struct{})
	if run == s.previewRun {
		s.previewsReady[imageIndex] = ready
	}
	return ready
}

// store renders the screen version of one capture first, so the kiosk can show it early, then composites the overlay
// and writes the capture. previewReady is closed once the screen version exists or failed.
func (s *CaptureService) store(ctx context.Context, run, imageIndex int, data *[]byte, previewReady chan struct{}) error {
	started := time.Now()

	base, err := jpeg.Decode(bytes.NewReader(*data))
	if err != nil {
		close(previewReady)
		return fmt.Errorf("service: decoding run %d image %d: %w", run, imageIndex, err)
	}

	preview, err := s.renderPreview(ctx, base, true)
	if err != nil {
		s.logger.Warn().Err(err).Int("run", run).Int("image", imageIndex).Msg("rendering the screen version failed")
	}
	s.publishPreview(run, imageIndex, preview)
	close(previewReady)
	previewAt := time.Now()

	composited, overlaid, err := s.compositeOverlay(ctx, base, data)
	if err != nil {
		return fmt.Errorf("service: compositing overlay for run %d image %d: %w", run, imageIndex, err)
	}
	compositedAt := time.Now()

	// A capture retried after a failure replaces the files the failed attempt left behind
	if err := s.captures.Delete(ctx, run, imageIndex); err != nil {
		return fmt.Errorf("service: replacing run %d image %d: %w", run, imageIndex, err)
	}

	// The original is only worth keeping separately when an overlay was actually applied,
	// without one the stored capture already is the unmodified image.
	var original []byte
	if s.options.SaveOriginals && overlaid {
		original = *data
	}
	if err := s.captures.Create(ctx, domain.Capture{
		Event:      s.eventSlug,
		Prefix:     s.options.FilePrefix,
		Run:        run,
		ImageIndex: imageIndex,
		TakenAt:    time.Now(),
	}, *composited, original); err != nil {
		return fmt.Errorf("service: storing run %d image %d: %w", run, imageIndex, err)
	}

	s.logger.Info().
		Int("run", run).
		Int("image", imageIndex).
		Int64("preview_ms", previewAt.Sub(started).Milliseconds()).
		Int64("composite_ms", compositedAt.Sub(previewAt).Milliseconds()).
		Int64("save_ms", time.Since(compositedAt).Milliseconds()).
		Msg("capture stored")

	return nil
}

// renderPreview scales base down to the screen version and encodes it, drawing the overlay in if withOverlay is set.
// The approximate bilinear scaler is enough for the screen and much faster than the one for the stored captures.
func (s *CaptureService) renderPreview(ctx context.Context, base image.Image, withOverlay bool) ([]byte, error) {
	bounds := previewBounds(base.Bounds())
	preview := image.NewRGBA(bounds)
	xdraw.ApproxBiLinear.Scale(preview, bounds, base, base.Bounds(), xdraw.Src, nil)

	if withOverlay && s.options.OverlayPath != "" {
		scaled, err := s.scaledOverlay(ctx, bounds)
		if err == nil {
			draw.Draw(preview, bounds, scaled, bounds.Min, draw.Over)
		}
	}

	encoded, err := media.EncodeJPEG(preview, previewQuality)
	if err != nil {
		return nil, fmt.Errorf("service: encoding screen version: %w", err)
	}
	return encoded, nil
}

// publishPreview keeps the screen version of a capture of the current run, nil keeps nothing.
func (s *CaptureService) publishPreview(run, imageIndex int, preview []byte) {
	if preview == nil {
		return
	}

	s.previewMu.Lock()
	defer s.previewMu.Unlock()

	if run == s.previewRun {
		s.previews[imageIndex] = preview
	}
}

// compositeOverlay draws the overlay over the decoded base of capture and returns the result as JPEG,
// together with whether an overlay was applied. Without an overlay, capture is returned unchanged.
// An overlay that cannot be read is logged and skipped, so it never stops the booth from taking photos.
func (s *CaptureService) compositeOverlay(ctx context.Context, base image.Image, capture *[]byte) (*[]byte, bool, error) {
	if s.options.OverlayPath == "" {
		return capture, false, nil
	}

	scaled, err := s.scaledOverlay(ctx, base.Bounds())
	if err != nil {
		s.logger.Warn().Err(err).Str("overlay", s.options.OverlayPath).Msg("storing capture without overlay")
		return capture, false, nil
	}

	bounds := base.Bounds()
	composited := image.NewRGBA(bounds)
	draw.Draw(composited, bounds, base, bounds.Min, draw.Src)
	draw.Draw(composited, bounds, scaled, bounds.Min, draw.Over)

	encoded, err := media.EncodeJPEG(composited, captureQuality)
	if err != nil {
		return nil, false, fmt.Errorf("encoding composited capture: %w", err)
	}

	return new(withExif(encoded, exifSegments(*capture))), true, nil
}

// jpegMarkerSOS starts the compressed image data, after which no metadata segment follows.
const jpegMarkerSOS = 0xDA

// exifHeader starts the payload of an EXIF segment, followed by a TIFF structure.
var exifHeader = []byte("Exif\x00\x00")

// exifSegments returns the EXIF segments of a JPEG, including their markers, or nil if it has none.
// Encoding a JPEG writes no metadata, so the composited capture gets the segments of the camera image.
func exifSegments(data []byte) []byte {
	var segments []byte

	// Every segment before the image data is a marker, a two-byte length, and the payload
	for i := len(jpegSOI); i+4 <= len(data) && data[i] == 0xFF; {
		marker := data[i+1]
		if marker == jpegMarkerSOS {
			break
		}

		end := i + 2 + int(data[i+2])<<8 + int(data[i+3])
		if end > len(data) {
			break
		}
		if marker == 0xE1 && bytes.HasPrefix(data[i+4:end], exifHeader) {
			segments = append(segments, withoutThumbnail(data[i:end])...)
		}
		i = end
	}

	return segments
}

// withoutThumbnail returns a copy of an EXIF segment without its thumbnail. The thumbnail lives in the second directory
// of the TIFF structure, which the end of the first one links to, so clearing that link leaves it unreferenced.
// It would show the photo without the overlay. A segment it cannot read stays as it is.
func withoutThumbnail(segment []byte) []byte {
	result := bytes.Clone(segment)

	// The TIFF structure starts after the marker, the length, and the EXIF header
	tiff := result[4+len(exifHeader):]
	if len(tiff) < 8 {
		return result
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return result
	}

	first := int(order.Uint32(tiff[4:8]))
	if first+2 > len(tiff) {
		return result
	}
	// Every directory entry takes 12 bytes, the link to the next directory follows the entries
	link := first + 2 + 12*int(order.Uint16(tiff[first:first+2]))
	if link+4 > len(tiff) {
		return result
	}
	order.PutUint32(tiff[link:link+4], 0)

	return result
}

// withExif inserts the EXIF segments right after the start marker of a JPEG.
func withExif(data, segments []byte) []byte {
	if len(segments) == 0 {
		return data
	}

	result := make([]byte, 0, len(data)+len(segments))
	result = append(result, data[:len(jpegSOI)]...)
	result = append(result, segments...)
	return append(result, data[len(jpegSOI):]...)
}

// RunStart is returned when a new run is started.
type RunStart struct {
	Run          int
	ImagesPerRun int
}

// CreateRun opens the next run for owner, the client that takes its captures.
// A run that owner left incomplete is discarded, and a run of another client blocks the start
// with ErrRunBusy until it completes or has been idle for runIdleTimeout.
func (s *CaptureService) CreateRun(ctx context.Context, owner string) (RunStart, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.awaitStoreLocked(ctx); err != nil {
		return RunStart{}, err
	}
	// A failed store belongs to the run that ends here, the new run starts without it
	s.storeErr = nil

	if s.openRun != 0 {
		if s.openOwner != owner && time.Since(s.lastActivity) < runIdleTimeout {
			return RunStart{}, ErrRunBusy
		}

		if err := s.runs.Delete(ctx, s.openRun); err != nil {
			return RunStart{}, fmt.Errorf("service: discarding incomplete run %d: %w", s.openRun, err)
		}
		s.openRun = 0
	}

	run, err := s.runs.Create(ctx)
	if err != nil {
		return RunStart{}, fmt.Errorf("service: determining next run: %w", err)
	}

	s.openRun = run
	s.openOwner = owner
	s.nextIndex = 1
	s.lastActivity = time.Now()
	s.resetPreviews(run)

	return RunStart{
		Run:          run,
		ImagesPerRun: s.options.ImagesPerRun,
	}, nil
}

// resetPreviews drops the screen versions of the previous run when run starts.
func (s *CaptureService) resetPreviews(run int) {
	s.previewMu.Lock()
	defer s.previewMu.Unlock()

	s.previewRun = run
	s.previews = make(map[int][]byte)
	s.previewsReady = make(map[int]chan struct{})
}

// AbortRun deletes every file stored for a run and, if it is the currently open run,
// closes it so its number can never accept further captures.
func (s *CaptureService) AbortRun(ctx context.Context, run int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.awaitStoreLocked(ctx); err != nil {
		return err
	}

	if run == s.openRun {
		s.openRun = 0
	}

	if err := s.runs.Delete(ctx, run); err != nil {
		return fmt.Errorf("service: deleting run %d: %w", run, err)
	}

	return nil
}

// awaitStoreLocked waits until the capture stored in the background is written.
// Callers hold s.mu, so no new store can start meanwhile.
func (s *CaptureService) awaitStoreLocked(ctx context.Context) error {
	if s.stored == nil {
		return nil
	}

	select {
	case <-s.stored:
		s.stored = nil
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PrepareOverlay scales the overlay to the fixed capture size and to its screen version ahead of the first capture,
// which would otherwise pay for it. Without an overlay or a fixed capture size, there is nothing to prepare.
func (s *CaptureService) PrepareOverlay(ctx context.Context) {
	if s.options.OverlayPath == "" || s.options.CaptureWidth <= 0 || s.options.CaptureHeight <= 0 {
		return
	}

	started := time.Now()
	captureBounds := image.Rect(0, 0, s.options.CaptureWidth, s.options.CaptureHeight)
	for _, bounds := range []image.Rectangle{captureBounds, previewBounds(captureBounds)} {
		if _, err := s.scaledOverlay(ctx, bounds); err != nil {
			s.logger.Warn().Err(err).Str("overlay", s.options.OverlayPath).Msg("preparing the overlay failed")
			return
		}
	}
	s.logger.Info().Int64("prepare_ms", time.Since(started).Milliseconds()).Msg("overlay prepared")
}

// previewBounds returns the largest size with the aspect ratio of bounds that fits previewWidth and previewHeight,
// and never enlarges a smaller image.
func previewBounds(bounds image.Rectangle) image.Rectangle {
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return image.Rect(0, 0, 1, 1)
	}

	scale := min(float64(previewWidth)/float64(width), float64(previewHeight)/float64(height), 1)
	return image.Rect(0, 0, max(1, int(float64(width)*scale)), max(1, int(float64(height)*scale)))
}

// overlayCache holds the decoded overlay and its copies scaled to the sizes of the captures and their screen versions,
// since the overlay and these sizes stay the same for the lifetime of the process.
type overlayCache struct {
	image  image.Image
	scaled map[image.Rectangle]*image.RGBA
}

// scaledOverlay returns the overlay scaled to bounds and caches it per size. The sharp but slow CatmullRom scaler
// keeps the edges of text clean, and the cache pays its cost once. A failed read is not cached.
func (s *CaptureService) scaledOverlay(ctx context.Context, bounds image.Rectangle) (*image.RGBA, error) {
	s.overlayMu.Lock()
	defer s.overlayMu.Unlock()

	if s.overlay != nil {
		if scaled, ok := s.overlay.scaled[bounds]; ok {
			return scaled, nil
		}
	}

	if s.overlay == nil {
		data, err := s.overlays.Get(ctx, s.options.OverlayPath)
		if err != nil {
			return nil, fmt.Errorf("reading overlay: %w", err)
		}

		decoded, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("decoding overlay: %w", err)
		}

		s.overlay = &overlayCache{
			image:  decoded,
			scaled: make(map[image.Rectangle]*image.RGBA),
		}
	}

	scaled := image.NewRGBA(bounds)
	xdraw.CatmullRom.Scale(scaled, bounds, s.overlay.image, s.overlay.image.Bounds(), xdraw.Src, nil)
	s.overlay.scaled[bounds] = scaled

	return scaled, nil
}

// Wait blocks until the captures stored in the background are written, for a clean end of the process.
func (s *CaptureService) Wait() {
	s.storing.Wait()
}
