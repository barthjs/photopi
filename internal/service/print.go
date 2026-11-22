package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
	"golang.org/x/sync/singleflight"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/store"
)

// lpstatEnv makes the CUPS tools answer in English, which the parsers of their output expect.
var lpstatEnv = []string{"LC_ALL=C"}

const (
	renderDPI = 300
	mmPerInch = 25.4
	ptPerInch = 72

	// maxPendingPrintJobs is the number of jobs a queue may hold before a new print is refused,
	// so guests do not stack up prints behind a stuck or slow printer.
	maxPendingPrintJobs = 3
	// collageRenderTimeout bounds one shared rendering of the collage of a run.
	collageRenderTimeout = 2 * time.Minute
	// layoutHashLength is the number of hex characters of the layout hash in a collage file name.
	layoutHashLength = 8
	// collageQuality is the JPEG quality of a rendered collage page.
	collageQuality = 90
)

// ErrPrintQueueUnavailable means the printer queue cannot take a job right now.
var ErrPrintQueueUnavailable = errors.New("service: the printer queue does not accept jobs")

// PrintSubmissionMode selects how print jobs reach CUPS.
type PrintSubmissionMode string

const (
	// PrintSubmissionManaged sends the PDF through the CUPS filters and the driver of the printer,
	// which scale it to the requested paper.
	PrintSubmissionManaged PrintSubmissionMode = "managed"
	// PrintSubmissionRaw sends the PDF untouched, so its page size alone determines the output.
	// It needs a printer with its own PDF interpreter.
	PrintSubmissionRaw PrintSubmissionMode = "raw"
)

// PrintOptions holds the settings of a PrintService.
type PrintOptions struct {
	Enabled bool
	// PrinterName is the CUPS queue (see lpstat -p).
	PrinterName string
	// Layout is the name of the layout that composes print and share collages.
	Layout         string
	SubmissionMode PrintSubmissionMode
	// EventName is drawn onto collages that carry branding and names the folder of the collages.
	EventName string
	// FilePrefix starts the file name of every collage, like the captures of a run.
	FilePrefix string
}

// PrintService renders collages of runs and submits them to a CUPS printer.
type PrintService struct {
	logger    zerolog.Logger
	options   PrintOptions
	commands  CommandRunner
	runs      *store.RunStore
	captures  *store.CaptureStore
	collages  *store.CollageStore
	layouts   *store.LayoutStore
	eventSlug string

	// renders lets concurrent print, mail, and cloud requests of one run share a single rendering.
	renders singleflight.Group
	jobs    *runJobs
}

// NewPrintService creates a PrintService.
func NewPrintService(
	logger zerolog.Logger,
	options PrintOptions,
	commands CommandRunner,
	runs *store.RunStore,
	captures *store.CaptureStore,
	collages *store.CollageStore,
	layouts *store.LayoutStore,
) *PrintService {
	return &PrintService{
		logger:    logger,
		options:   options,
		commands:  commands,
		runs:      runs,
		captures:  captures,
		collages:  collages,
		layouts:   layouts,
		eventSlug: slugify(options.EventName),
		jobs:      newRunJobs(),
	}
}

// PrinterInfo describes one CUPS queue as reported by lpstat.
type PrinterInfo struct {
	Name   string
	Status PrinterStatus
}

// PrinterStatus is the state of a CUPS queue as lpstat -p reports it.
type PrinterStatus string

const (
	PrinterIdle     PrinterStatus = "idle"
	PrinterPrinting PrinterStatus = "printing"
	PrinterDisabled PrinterStatus = "disabled"
	// PrinterHolding is an enabled queue that holds new jobs instead of printing them.
	PrinterHolding PrinterStatus = "holding"
)

// ListPrinters returns every CUPS queue known to lpstat. No queue gives an empty list, not an error.
func (s *PrintService) ListPrinters(ctx context.Context) ([]PrinterInfo, error) {
	out, err := s.commands.Output(ctx, lpstatEnv, "lpstat", "-p")
	if err == nil {
		return parsePrinterStatus(out), nil
	}

	exitErr, ok := errors.AsType[*exec.ExitError](err)
	if !ok {
		return nil, fmt.Errorf("service: running lpstat -p: %w", err)
	}

	message := strings.TrimSpace(string(exitErr.Stderr) + "\n" + string(out))
	if strings.Contains(message, "No destinations added") {
		return nil, nil
	}
	return nil, fmt.Errorf("service: running lpstat -p: %w (%s)", err, message)
}

// lpstatLinePattern matches the status line of one queue in the output of lpstat -p.
var lpstatLinePattern = regexp.MustCompile(`^printer\s+(\S+)\s+(is idle|now printing|disabled since|is holding new jobs)\b`)

// lpstatStatuses maps the wording of lpstat -p onto the queue status.
var lpstatStatuses = map[string]PrinterStatus{
	"is idle":             PrinterIdle,
	"now printing":        PrinterPrinting,
	"disabled since":      PrinterDisabled,
	"is holding new jobs": PrinterHolding,
}

// Get returns the state of the latest print of a run, found is false if no print was started for it.
func (s *PrintService) Get(run int) (state JobState, found bool) {
	return s.jobs.get(run)
}

// Enabled reports whether physical printing is turned on in the configuration.
func (s *PrintService) Enabled() bool {
	return s.options.Enabled
}

// Layout returns the layout the running service prints with.
func (s *PrintService) Layout() string {
	return s.options.Layout
}

// RenderCollage composes the captures of run onto the pages of the configured layout
// and returns the storage path of each rendered page in order.
// The pages are cached until the layout changes, so every share reuses them.
func (s *PrintService) RenderCollage(ctx context.Context, run int) ([]string, error) {
	layout, err := s.layout(ctx)
	if err != nil {
		return nil, err
	}

	return s.renderCollage(ctx, run, *layout)
}

// filledPage is a page of a layout that at least one capture of a run reaches.
type filledPage struct {
	index int
	// slots are the slots of the page that receive a capture.
	slots []domain.LayoutSlot
	// firstCapture is the index of the capture that fills the first slot.
	firstCapture int
}

// TestConnection reports whether printerName is a CUPS queue that prints new jobs, without printing anything.
func (s *PrintService) TestConnection(ctx context.Context, printerName string) (ok bool, message string) {
	if printerName == "" {
		return false, "no printer selected"
	}

	out, err := s.commands.CombinedOutput(ctx, lpstatEnv, "lpstat", "-p", queueName(printerName))
	if err != nil {
		return false, strings.TrimSpace(string(out))
	}

	printer, found := findPrinter(parsePrinterStatus(out), queueName(printerName))
	if !found {
		return false, fmt.Sprintf("printer %q not found", printerName)
	}
	switch printer.Status {
	case PrinterDisabled:
		return false, fmt.Sprintf("printer %q is disabled", printerName)
	case PrinterHolding:
		return false, fmt.Sprintf("printer %q is holding new jobs", printerName)
	case PrinterIdle, PrinterPrinting:
	}

	return true, fmt.Sprintf("printer %q is %s", printerName, printer.Status)
}

// Begin marks the print of a run as pending and reports whether the caller should start it,
// which is false while a print of the run is already in flight.
func (s *PrintService) Begin(run int) bool {
	return s.jobs.begin(run)
}

// Print renders the collage pages of run and submits them to the configured queue as one PDF job.
// The outcome is recorded for Get.
func (s *PrintService) Print(ctx context.Context, run int) (err error) {
	defer func() {
		s.jobs.finish(run, printJobState(err))
	}()

	if !s.options.Enabled {
		return errors.New("service: printing is not enabled")
	}
	if s.options.PrinterName == "" {
		return errors.New("service: no printer configured")
	}

	if err := s.CheckQueue(ctx); err != nil {
		return err
	}

	layout, err := s.layout(ctx)
	if err != nil {
		return err
	}

	renderedPaths, err := s.renderCollage(ctx, run, *layout)
	if err != nil {
		return fmt.Errorf("service: rendering collage for run %d: %w", run, err)
	}

	pages := make([][]byte, 0, len(renderedPaths))
	for _, renderedPath := range renderedPaths {
		data, err := s.runs.GetFile(ctx, renderedPath)
		if err != nil {
			return fmt.Errorf("service: reading collage %q: %w", renderedPath, err)
		}
		pages = append(pages, data)
	}

	pdf, err := wrapJPEGsAsPDF(pages, layout.Paper.WidthMM, layout.Paper.HeightMM)
	if err != nil {
		return fmt.Errorf("service: building print document for run %d: %w", run, err)
	}

	jobID, err := s.submitPrintJob(ctx, pdf, layout.Paper)
	if err != nil {
		return fmt.Errorf("service: submitting print job for run %d: %w", run, err)
	}

	s.logger.Info().Int("run", run).Str("printer", s.options.PrinterName).Str("job", jobID).Int("pages", len(pages)).Msg("print job submitted")

	return nil
}

// layout returns the configured layout, or apperr.ErrNotFound if it does not exist
// or its ID cannot address a layout file.
func (s *PrintService) layout(ctx context.Context) (*domain.Layout, error) {
	layout, err := s.layouts.Get(ctx, s.options.Layout)
	if errors.Is(err, store.ErrInvalidName) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("service: loading layout %q: %w", s.options.Layout, err)
	}
	return layout, nil
}

// renderCollage renders the collage of run with layout once for all concurrent callers of the run.
// The shared rendering ignores the cancellation of the first caller, so it cannot fail the other callers.
func (s *PrintService) renderCollage(ctx context.Context, run int, layout domain.Layout) ([]string, error) {
	result, err, _ := s.renders.Do(strconv.Itoa(run), func() (any, error) {
		renderCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), collageRenderTimeout)
		defer cancel()

		return s.collagePages(renderCtx, run, layout)
	})
	if err != nil {
		return nil, err
	}

	paths, ok := result.([]string)
	if !ok {
		return nil, fmt.Errorf("service: unexpected collage result %T", result)
	}
	return paths, nil
}

// collagePages returns the cached pages of run for layout, or renders all of them again if any is missing,
// after removing the collages that older layouts left behind.
func (s *PrintService) collagePages(ctx context.Context, run int, layout domain.Layout) ([]string, error) {
	capturePaths, err := s.captures.ListByRun(ctx, run, false)
	if err != nil {
		return nil, fmt.Errorf("service: listing run %d captures: %w", run, err)
	}
	if len(capturePaths) == 0 {
		return nil, apperr.ErrNotFound
	}

	hash, err := layoutHash(layout, s.options.EventName)
	if err != nil {
		return nil, err
	}

	pages := filledPages(layout, len(capturePaths))
	if len(pages) == 0 {
		return nil, fmt.Errorf("service: layout %q has no slot to fill", layout.Name)
	}

	collages := make([]domain.Collage, 0, len(pages))
	renderedPaths := make([]string, 0, len(pages))
	for _, page := range pages {
		collage := domain.Collage{
			Event:      s.eventSlug,
			Prefix:     s.options.FilePrefix,
			Run:        run,
			Page:       page.index,
			LayoutHash: hash,
		}
		collages = append(collages, collage)
		renderedPaths = append(renderedPaths, s.collages.Path(collage))
	}

	cached, err := s.collages.Exists(ctx, collages)
	if err != nil {
		return nil, fmt.Errorf("service: checking collage cache for run %d: %w", run, err)
	}
	if cached {
		return renderedPaths, nil
	}

	if err := s.collages.Delete(ctx, run); err != nil {
		return nil, fmt.Errorf("service: removing old collages of run %d: %w", run, err)
	}

	widthPx := mmToPixels(layout.Paper.WidthMM)
	heightPx := mmToPixels(layout.Paper.HeightMM)

	// Parsing the font is the expensive part of the branding and the same for every page
	var brander *brandingRenderer
	if layout.Branding.Enabled {
		brander, err = newBrandingRenderer(layout.Branding)
		if err != nil {
			return nil, fmt.Errorf("service: preparing branding: %w", err)
		}
		defer brander.Close()
	}

	for i, page := range pages {
		canvas := image.NewRGBA(image.Rect(0, 0, widthPx, heightPx))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

		for slotIndex, slot := range page.slots {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			capturePath := capturePaths[page.firstCapture+slotIndex]
			data, err := s.runs.GetFile(ctx, capturePath)
			if err != nil {
				return nil, fmt.Errorf("service: reading %q: %w", capturePath, err)
			}

			dstRect := image.Rect(
				int(slot.X*float64(widthPx)),
				int(slot.Y*float64(heightPx)),
				int((slot.X+slot.W)*float64(widthPx)),
				int((slot.Y+slot.H)*float64(heightPx)),
			)
			if err := drawPhoto(canvas, dstRect, data); err != nil {
				return nil, fmt.Errorf("service: drawing %q: %w", capturePath, err)
			}
		}

		if brander != nil {
			if err := brander.draw(canvas, s.options.EventName); err != nil {
				return nil, fmt.Errorf("service: rendering branding on page %d: %w", page.index, err)
			}
		}

		encoded, err := media.EncodeJPEG(canvas, collageQuality)
		if err != nil {
			return nil, fmt.Errorf("service: encoding collage for run %d page %d: %w", run, page.index, err)
		}

		if err := s.collages.Create(ctx, collages[i], encoded); err != nil {
			return nil, err
		}
	}

	return renderedPaths, nil
}

// layoutHash returns a short hash of the layout and the event name its caption shows,
// which keys the collage cache, so a changed layout or event name renders new pages.
func layoutHash(layout domain.Layout, eventName string) (string, error) {
	data, err := json.Marshal(layout)
	if err != nil {
		return "", fmt.Errorf("service: hashing layout %q: %w", layout.Name, err)
	}

	sum := sha256.New()
	_, _ = sum.Write(data)
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write([]byte(eventName))
	return hex.EncodeToString(sum.Sum(nil))[:layoutHashLength], nil
}

// filledPages distributes captures over the pages of layout in order and leaves out pages without a capture.
func filledPages(layout domain.Layout, captures int) []filledPage {
	var pages []filledPage
	next := 0
	for index, page := range layout.Pages {
		if next >= captures {
			break
		}

		count := min(len(page.Slots), captures-next)
		if count == 0 {
			continue
		}

		pages = append(pages, filledPage{
			index:        index,
			slots:        page.Slots[:count],
			firstCapture: next,
		})
		next += count
	}

	return pages
}

// drawPhoto decodes a capture, turns it upright by its EXIF orientation,
// crops it from the center to the aspect ratio of dstRect, and scales it into dstRect of canvas.
func drawPhoto(canvas *image.RGBA, dstRect image.Rectangle, data []byte) error {
	photo, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decoding: %w", err)
	}
	if dstRect.Empty() {
		return nil
	}

	orientation := jpegOrientation(data)
	// Orientations 5 to 8 turn the photo by 90 degrees, so it is cropped to the turned aspect ratio
	turned := orientation >= 5
	aspect := float64(dstRect.Dx()) / float64(dstRect.Dy())
	if turned {
		aspect = 1 / aspect
	}
	srcRect := centerCrop(photo.Bounds(), aspect)

	if orientation == 1 {
		xdraw.CatmullRom.Scale(canvas, dstRect, photo, srcRect, xdraw.Over, nil)
		return nil
	}

	// Scaling first and turning the small result is far cheaper than turning the full capture
	scaledSize := dstRect.Size()
	if turned {
		scaledSize = image.Pt(scaledSize.Y, scaledSize.X)
	}
	scaled := image.NewRGBA(image.Rectangle{
		Max: scaledSize,
	})
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), photo, srcRect, xdraw.Src, nil)

	upright := orientRGBA(scaled, orientation)
	draw.Draw(canvas, dstRect, upright, image.Point{}, draw.Over)
	return nil
}

// jpegOrientation returns the EXIF orientation (1 to 8) of JPEG data, or 1 if it carries none.
// It reads only the orientation tag of the first image directory of the APP1 segment.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}

	offset := 2
	for offset+4 <= len(data) {
		if data[offset] != 0xFF {
			return 1
		}
		marker := data[offset+1]
		// Start of scan or end of image, the metadata segments are over
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}

		length := int(binary.BigEndian.Uint16(data[offset+2:]))
		end := offset + 2 + length
		if length < 2 || end > len(data) {
			return 1
		}

		segment := data[offset+4 : end]
		if marker == 0xE1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			return exifOrientation(segment[6:])
		}
		offset = end
	}

	return 1
}

// exifOrientation reads the orientation tag from the first image directory of a TIFF structure.
func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:]) != 42 {
		return 1
	}

	const orientationTag = 0x0112
	const entrySize = 12

	directory := int(order.Uint32(tiff[4:]))
	if directory < 8 || directory+2 > len(tiff) {
		return 1
	}

	entries := int(order.Uint16(tiff[directory:]))
	for i := range entries {
		entry := directory + 2 + i*entrySize
		if entry+entrySize > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[entry:]) != orientationTag {
			continue
		}

		orientation := int(order.Uint16(tiff[entry+8:]))
		if orientation < 1 || orientation > 8 {
			return 1
		}
		return orientation
	}

	return 1
}

// centerCrop returns the largest rectangle with the aspect ratio (width / height) in the center of bounds.
func centerCrop(bounds image.Rectangle, aspect float64) image.Rectangle {
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 || aspect <= 0 {
		return bounds
	}

	cropWidth, cropHeight := width, height
	if float64(width)/float64(height) > aspect {
		cropWidth = max(1, int(math.Round(float64(height)*aspect)))
	} else {
		cropHeight = max(1, int(math.Round(float64(width)/aspect)))
	}

	minX := bounds.Min.X + (width-cropWidth)/2
	minY := bounds.Min.Y + (height-cropHeight)/2
	return image.Rect(minX, minY, minX+cropWidth, minY+cropHeight)
}

// orientRGBA turns an image as stored with the EXIF orientation into its upright form.
func orientRGBA(src *image.RGBA, orientation int) *image.RGBA {
	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	dstWidth, dstHeight := width, height
	if orientation >= 5 {
		dstWidth, dstHeight = height, width
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	for y := range dstHeight {
		for x := range dstWidth {
			srcX, srcY := orientedSource(orientation, x, y, width, height)
			srcOffset := src.PixOffset(src.Bounds().Min.X+srcX, src.Bounds().Min.Y+srcY)
			dstOffset := dst.PixOffset(x, y)
			copy(dst.Pix[dstOffset:dstOffset+4], src.Pix[srcOffset:srcOffset+4])
		}
	}

	return dst
}

// orientedSource returns the pixel of the stored image of width x height
// that lands on x, y of the upright image for an EXIF orientation.
func orientedSource(orientation int, x, y, width, height int) (int, int) {
	switch orientation {
	case 2:
		return width - 1 - x, y
	case 3:
		return width - 1 - x, height - 1 - y
	case 4:
		return x, height - 1 - y
	case 5:
		return y, x
	case 6:
		return y, height - 1 - x
	case 7:
		return width - 1 - y, height - 1 - x
	case 8:
		return width - 1 - y, x
	default:
		return x, y
	}
}

// printJobState maps the outcome of a print onto the state reported to the guest.
func printJobState(err error) JobState {
	if err == nil {
		return JobState{
			Status: JobDone,
		}
	}
	if errors.Is(err, ErrPrintQueueUnavailable) {
		return JobState{
			Status: JobFailed,
			Reason: JobReasonQueueUnavailable,
		}
	}
	return JobState{
		Status: JobFailed,
		Reason: JobReasonFailed,
	}
}

// CheckQueue returns ErrPrintQueueUnavailable if the configured queue cannot take a job right now.
func (s *PrintService) CheckQueue(ctx context.Context) error {
	name := queueName(s.options.PrinterName)

	out, err := s.commands.Output(ctx, lpstatEnv, "lpstat", "-p", name)
	if err != nil {
		return fmt.Errorf("%w: lpstat -p %s: %w", ErrPrintQueueUnavailable, name, err)
	}

	printer, found := findPrinter(parsePrinterStatus(out), name)
	if !found {
		return fmt.Errorf("%w: printer %q not found", ErrPrintQueueUnavailable, name)
	}
	switch printer.Status {
	case PrinterDisabled, PrinterHolding:
		return fmt.Errorf("%w: printer %q is %s", ErrPrintQueueUnavailable, name, printer.Status)
	case PrinterIdle, PrinterPrinting:
	}

	out, err = s.commands.Output(ctx, lpstatEnv, "lpstat", "-o", name)
	if err != nil {
		return fmt.Errorf("%w: lpstat -o %s: %w", ErrPrintQueueUnavailable, name, err)
	}

	pending := countQueuedJobs(out, name)
	if pending > maxPendingPrintJobs {
		return fmt.Errorf("%w: printer %q holds %d jobs", ErrPrintQueueUnavailable, name, pending)
	}

	return nil
}

// parsePrinterStatus parses the status line of every queue in the output of lpstat -p and ignores other lines.
func parsePrinterStatus(output []byte) []PrinterInfo {
	var printers []PrinterInfo
	for line := range strings.SplitSeq(string(output), "\n") {
		match := lpstatLinePattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		printers = append(printers, PrinterInfo{
			Name:   match[1],
			Status: lpstatStatuses[match[2]],
		})
	}

	return printers
}

// countQueuedJobs counts the jobs of a queue in the output of lpstat -o,
// which lists one job per line, starting with its ID "<queue>-<number>".
func countQueuedJobs(output []byte, name string) int {
	count := 0
	for line := range strings.SplitSeq(string(output), "\n") {
		if strings.HasPrefix(line, name+"-") {
			count++
		}
	}
	return count
}

// submitPrintJob submits a PDF to the configured queue as one lp job and returns the job ID that lp reports.
// The managed mode asks the driver for the paper of the layout and to fill it, the raw mode leaves the PDF untouched.
func (s *PrintService) submitPrintJob(ctx context.Context, pdf []byte, paper domain.LayoutPaper) (string, error) {
	tmpPath, err := writeTempFile("photopi-print-*.pdf", pdf)
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmpPath) }()

	args := []string{"-d", s.options.PrinterName}
	switch s.options.SubmissionMode {
	case PrintSubmissionRaw:
		args = append(args, "-o", "raw")
	default:
		mediaOption := fmt.Sprintf("media=Custom.%.2fx%.2fin", paper.WidthMM/mmPerInch, paper.HeightMM/mmPerInch)
		args = append(args, "-o", mediaOption, "-o", "print-scaling=fill")
	}
	args = append(args, tmpPath)

	out, err := s.commands.CombinedOutput(ctx, lpstatEnv, "lp", args...)
	if err != nil {
		return "", fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
	}

	return parseJobID(out), nil
}

// lpRequestPattern matches the job ID in the answer of lp, "request id is <queue>-<number> (1 file(s))".
var lpRequestPattern = regexp.MustCompile(`request id is (\S+)`)

// parseJobID returns the job ID that lp reports, or an empty string if its answer holds none.
func parseJobID(output []byte) string {
	match := lpRequestPattern.FindSubmatch(output)
	if match == nil {
		return ""
	}
	return string(match[1])
}

// writeTempFile writes data to a new temp file matching pattern and returns its path.
// It leaves no file behind on failure.
func writeTempFile(pattern string, data []byte) (string, error) {
	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	return tmpPath, nil
}

// wrapJPEGsAsPDF embeds JPEG collage pages as the pages of a minimal PDF with the physical size of the paper.
func wrapJPEGsAsPDF(pages [][]byte, widthMM, heightMM float64) ([]byte, error) {
	if len(pages) == 0 {
		return nil, errors.New("no page to print")
	}

	pageWidthPt := widthMM / mmPerInch * ptPerInch
	pageHeightPt := heightMM / mmPerInch * ptPerInch

	// The PDF is built by hand, since full-page JPEG images need only a small, fixed structure.
	// Objects: 1 catalog, 2 page tree, 3 the content stream all pages share,
	// then a page object and its image for every page.
	const firstPageObject = 4
	kids := make([]string, 0, len(pages))
	for i := range pages {
		kids = append(kids, fmt.Sprintf("%d 0 R", firstPageObject+2*i))
	}

	var buf bytes.Buffer
	_, _ = buf.WriteString("%PDF-1.4\n")

	var offsets []int
	startObject := func() int {
		offsets = append(offsets, buf.Len())
		return len(offsets)
	}
	writeObject := func(body string) {
		number := startObject()
		_, _ = fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", number, body)
	}

	writeObject("<< /Type /Catalog /Pages 2 0 R >>")
	writeObject(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	content := fmt.Sprintf("q %.2f 0 0 %.2f 0 0 cm /Im0 Do Q", pageWidthPt, pageHeightPt)
	writeObject(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))

	for i, page := range pages {
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(page))
		if err != nil {
			return nil, fmt.Errorf("decoding JPEG header of page %d: %w", i, err)
		}

		imageObject := firstPageObject + 2*i + 1
		writeObject(fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /XObject << /Im0 %d 0 R >> >> /Contents 3 0 R >>",
			pageWidthPt,
			pageHeightPt,
			imageObject,
		))

		number := startObject()
		_, _ = fmt.Fprintf(
			&buf,
			"%d 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n",
			number,
			cfg.Width,
			cfg.Height,
			len(page),
		)
		_, _ = buf.Write(page)
		_, _ = buf.WriteString("\nendstream\nendobj\n")
	}

	xrefOffset := buf.Len()
	_, _ = fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, off := range offsets {
		_, _ = fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	_, _ = fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(offsets)+1, xrefOffset)

	return buf.Bytes(), nil
}

// Resume enables a queue again and releases its held jobs.
func (s *PrintService) Resume(ctx context.Context, printerName string) error {
	name, err := s.knownQueue(ctx, printerName)
	if err != nil {
		return err
	}

	out, err := s.commands.CombinedOutput(ctx, lpstatEnv, "cupsenable", "--release", name)
	if err != nil {
		return fmt.Errorf("service: resuming printer %q: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}

	s.logger.Info().Str("printer", name).Msg("printer queue resumed")
	return nil
}

// Cancel cancels every job of a queue.
func (s *PrintService) Cancel(ctx context.Context, printerName string) error {
	name, err := s.knownQueue(ctx, printerName)
	if err != nil {
		return err
	}

	out, err := s.commands.CombinedOutput(ctx, lpstatEnv, "cancel", "-a", name)
	if err != nil {
		return fmt.Errorf("service: canceling the jobs of printer %q: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}

	s.logger.Info().Str("printer", name).Msg("printer jobs canceled")
	return nil
}

// knownQueue returns the queue of printerName, or apperr.ErrNotFound if lpstat does not know it,
// so only an existing queue ever reaches a CUPS command.
func (s *PrintService) knownQueue(ctx context.Context, printerName string) (string, error) {
	printers, err := s.ListPrinters(ctx)
	if err != nil {
		return "", err
	}

	name := queueName(printerName)
	if _, found := findPrinter(printers, name); !found {
		return "", apperr.ErrNotFound
	}
	return name, nil
}

// queueName returns the CUPS queue of a destination, without the instance after a slash,
// since the queue commands only know the queue itself.
func queueName(printerName string) string {
	name, _, _ := strings.Cut(printerName, "/")
	return name
}

// findPrinter returns the queue named name from printers.
func findPrinter(printers []PrinterInfo, name string) (PrinterInfo, bool) {
	for _, printer := range printers {
		if printer.Name == name {
			return printer, true
		}
	}
	return PrinterInfo{}, false
}

// brandingRenderer draws the optional caption of a layout onto the canvas of a page.
// One instance is built per rendering and reused for every page, since parsing the font is the expensive part.
type brandingRenderer struct {
	template    string
	x, y        float64
	rotationDeg float64
	drawer      *font.Drawer
	face        font.Face
}

// newBrandingRenderer parses the branding font and prepares a reusable drawer for it.
// Close must be called once the caller is done rendering pages with it.
func newBrandingRenderer(branding domain.LayoutBranding) (*brandingRenderer, error) {
	fontSizePt := branding.FontSizePt
	if fontSizePt <= 0 {
		fontSizePt = domain.DefaultBrandingFontSizePt
	}

	textColor, err := parseHexColor(branding.Color)
	if err != nil {
		textColor = color.Black
	}

	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parsing branding font: %w", err)
	}

	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    fontSizePt,
		DPI:     renderDPI,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("creating branding font face: %w", err)
	}

	return &brandingRenderer{
		template:    branding.Template,
		x:           branding.X,
		y:           branding.Y,
		rotationDeg: branding.RotationDeg,
		drawer: &font.Drawer{
			Src:  image.NewUniform(textColor),
			Face: face,
		},
		face: face,
	}, nil
}

// Close releases the font face's resources.
func (b *brandingRenderer) Close() {
	_ = b.face.Close()
}

// draw renders the caption on a translucent box, so it stays legible on any photo,
// rotated by rotationDeg around its center at x, y. font.Drawer only draws horizontal text,
// so the caption is rendered into a sprite first and then rotated onto canvas.
func (b *brandingRenderer) draw(canvas *image.RGBA, eventName string) error {
	text := strings.NewReplacer("{{eventName}}", eventName, "{{date}}", time.Now().Format("02.01.2006")).Replace(b.template)
	if text == "" {
		return nil
	}

	textWidth := b.drawer.MeasureString(text).Round()
	metrics := b.face.Metrics()
	ascent, descent := metrics.Ascent.Round(), metrics.Descent.Round()

	const boxPaddingPx = 6
	spriteW := textWidth + 2*boxPaddingPx
	spriteH := ascent + descent + 2*boxPaddingPx
	if spriteW <= 0 || spriteH <= 0 {
		return nil
	}

	sprite := image.NewRGBA(image.Rect(0, 0, spriteW, spriteH))
	draw.Draw(sprite, sprite.Bounds(), image.NewUniform(color.NRGBA{
		R: 255,
		G: 255,
		B: 255,
		A: 190,
	}), image.Point{}, draw.Src)
	b.drawer.Dst = sprite
	b.drawer.Dot = fixed.Point26_6{
		X: fixed.I(boxPaddingPx),
		Y: fixed.I(boxPaddingPx + ascent),
	}
	b.drawer.DrawString(text)

	bounds := canvas.Bounds()
	centerX := b.x * float64(bounds.Dx())
	centerY := b.y * float64(bounds.Dy())
	spriteCenterX := float64(spriteW) / 2
	spriteCenterY := float64(spriteH) / 2

	angle := b.rotationDeg * math.Pi / 180
	cosA, sinA := math.Cos(angle), math.Sin(angle)

	// Rotate the sprite about its own center, then translate that center to (centerX, centerY),
	// expressed directly as the Aff3 matrix Transform expects:
	// src-space (sx,sy) maps to dst-space (m[0]*sx+m[1]*sy+m[2], m[3]*sx+m[4]*sy+m[5]).
	m := f64.Aff3{
		cosA, -sinA, centerX - spriteCenterX*cosA + spriteCenterY*sinA,
		sinA, cosA, centerY - spriteCenterX*sinA - spriteCenterY*cosA,
	}
	xdraw.CatmullRom.Transform(canvas, m, sprite, sprite.Bounds(), xdraw.Over, nil)

	return nil
}

// parseHexColor parses a "#rrggbb" string into an opaque color.
// It returns black with an error on failure, so an invalid color never breaks rendering.
func parseHexColor(hex string) (color.Color, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return color.Black, fmt.Errorf("branding: invalid color %q", hex)
	}

	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.Black, fmt.Errorf("branding: invalid color %q: %w", hex, err)
	}

	return color.RGBA{
		R: uint8(value >> 16),
		G: uint8(value >> 8),
		B: uint8(value),
		A: 255,
	}, nil
}

// mmToPixels converts millimeters to pixels at the render resolution.
func mmToPixels(mm float64) int {
	return int(mm / mmPerInch * renderDPI)
}
