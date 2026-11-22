package camera

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

// gracefulStopGrace bounds how long a camera process gets to exit cleanly after SIGTERM before it is killed,
// since a SIGKILL leaves the PTP session of gphoto2 half-open and the next invocation fails to claim the device.
const gracefulStopGrace = 2 * time.Second

// captureTimeout bounds a whole capture, from stopping the preview to receiving the still.
// A camera that has not delivered within this time counts as failed instead of blocking the run.
const captureTimeout = 20 * time.Second

// maxStderrBytes bounds how much of the most recent stderr output of a tool is kept for logs and errors.
const maxStderrBytes = 8 << 10

// command describes one invocation of a camera tool.
type command struct {
	// label names the invocation in logs and errors.
	label string
	name  string
	args  []string
	// env is the environment of the process, nil to inherit the one of this process.
	env []string
}

// streamPreviewProcess claims the device for a preview, runs the command,
// and forwards its raw MJPEG output to w as a multipart/x-mixed-replace stream
// until ctx is canceled, a capture stops the preview, or the tool stops producing frames.
func streamPreviewProcess(ctx context.Context, logger zerolog.Logger, device *deviceSession, spec command, w io.Writer, boundary string) error {
	previewCtx, finish, err := device.startPreview(ctx)
	if err != nil {
		return fmt.Errorf("camera: waiting for the device to start %s: %w", spec.label, err)
	}

	// The process gets its own context, so the cleanup can stop it after the stream ended on its own
	processCtx, stop := context.WithCancel(previewCtx)
	stderr := newTailBuffer(maxStderrBytes)
	cmd := spec.build(processCtx, stderr)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stop()
		finish()
		return fmt.Errorf("camera: creating %s stdout pipe: %w", spec.label, err)
	}

	if err := cmd.Start(); err != nil {
		stop()
		finish()
		return fmt.Errorf("camera: starting %s: %w", spec.label, err)
	}

	var streamErr error
	defer func() {
		stop()
		// The tool only exits once its pending writes to stdout complete,
		// and Wait must not run before the pipe is read to the end.
		// WaitDelay kills the process and closes the pipe if it hangs.
		_, _ = io.Copy(io.Discard, stdout)
		waitErr := cmd.Wait()
		// The device only counts as free once the process has exited
		finish()
		logPreviewExit(logger, spec.label, streamErr, waitErr, stderr.String())
	}()

	streamErr = streamMJPEGFrames(processCtx, stdout, w, boundary)
	// The preview context ends without the request context only when a capture claimed the device
	if errors.Is(streamErr, context.Canceled) && ctx.Err() == nil && previewCtx.Err() != nil {
		streamErr = ErrPreviewStopped
	}
	return streamErr
}

// logPreviewExit logs why a preview process ended, as a warning if the stream failed or the tool reported problems,
// since this is the only place that explains a failing preview.
func logPreviewExit(logger zerolog.Logger, label string, streamErr, waitErr error, stderr string) {
	failed := streamErr != nil && !isIntentionalStop(streamErr)

	event := logger.Debug()
	if failed || reportsProblems(stderr) {
		event = logger.Warn()
	}
	if failed {
		event = event.Err(streamErr).AnErr("exit", waitErr)
	}
	if stderr != "" {
		event = event.Str("stderr", stderr)
	}
	event.Msg(label + " exited")
}

// isIntentionalStop reports whether a preview stream ended because it was stopped
// or the client stopped reading, as opposed to the camera failing.
func isIntentionalStop(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrPreviewStopped) {
		return true
	}
	_, isWriteErr := errors.AsType[*previewWriteError](err)
	return isWriteErr
}

// zslHint is the hint rpicam-still prints for every --immediate capture.
// The booth takes its stills without the preview phase on purpose,
// so a temporal denoise is not available and the hint is no problem.
const zslHint = "WARNING: Capture will not make use of temporal denoise"

// captureProcess stops a running preview, waits until it has released the device,
// and runs the command once to take a still, all within captureTimeout. It logs how long both steps took.
func captureProcess(ctx context.Context, logger zerolog.Logger, device *deviceSession, spec command) (*[]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, captureTimeout)
	defer cancel()

	stopStarted := time.Now()
	release, err := device.stopAndWait(ctx)
	if err != nil {
		return nil, fmt.Errorf("camera: stopping the preview for %s: %w", spec.label, err)
	}
	defer release()
	stopDuration := time.Since(stopStarted)

	stderr := newTailBuffer(maxStderrBytes)
	cmd := spec.build(ctx, stderr)

	stillStarted := time.Now()
	output, err := cmd.Output()
	stillDuration := time.Since(stillStarted)

	logger.Info().
		Str("tool", spec.label).
		Int64("preview_stop_ms", stopDuration.Milliseconds()).
		Int64("still_ms", stillDuration.Milliseconds()).
		Bool("success", err == nil).
		Msg("camera capture")

	if err != nil {
		// A canceled or expired context stops the tool, which then exits with a plain signal error
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("camera: %s failed: %w: %w (%s)", spec.label, ctxErr, err, stderr.String())
		}
		return nil, fmt.Errorf("camera: %s failed: %w (%s)", spec.label, err, stderr.String())
	}

	if text := stderr.String(); text != "" {
		if reportsProblems(text) {
			logger.Warn().Str("stderr", text).Msg(spec.label + " reported warnings")
		} else {
			logger.Debug().Str("stderr", text).Msg(spec.label + " output")
		}
	}

	return &output, nil
}

// build creates the process for ctx, which a canceled ctx stops with SIGTERM and kills after gracefulStopGrace.
func (c command) build(ctx context.Context, stderr io.Writer) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.name, c.args...)
	cmd.Env = c.env
	cmd.Stderr = stderr
	// Overrides the default of SIGKILL on cancel, so the tool releases the device cleanly first
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = gracefulStopGrace
	return cmd
}

// reportsProblems reports whether stderr output of a tool contains a warning or an error,
// as opposed to the progress messages the tools also write there.
func reportsProblems(stderr string) bool {
	lower := strings.ToLower(withoutKnownHints(stderr))
	for _, word := range []string{"warn", "error", "fail"} {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

// withoutKnownHints removes the hints that the tools print on every run although nothing is wrong,
// together with their indented continuation lines.
func withoutKnownHints(stderr string) string {
	lines := strings.Split(stderr, "\n")
	kept := make([]string, 0, len(lines))
	inHint := false
	for _, line := range lines {
		if strings.HasPrefix(line, zslHint) {
			inHint = true
			continue
		}
		if inHint && strings.HasPrefix(line, " ") {
			continue
		}
		inHint = false
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// tailBuffer is an io.Writer that keeps only the last limit bytes written to it.
type tailBuffer struct {
	limit     int
	buf       []byte
	truncated bool
}

// newTailBuffer creates a tailBuffer that keeps at most limit bytes.
func newTailBuffer(limit int) *tailBuffer {
	return &tailBuffer{
		limit: limit,
	}
}

// Write appends p and drops the oldest bytes beyond the limit.
func (b *tailBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	if overflow := len(b.buf) - b.limit; overflow > 0 {
		b.buf = b.buf[:copy(b.buf, b.buf[overflow:])]
		b.truncated = true
	}
	return len(p), nil
}

// String returns the kept output without surrounding whitespace, starting at a full line if older output was dropped.
func (b *tailBuffer) String() string {
	text := string(b.buf)
	if b.truncated {
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			text = text[i+1:]
		}
	}
	return strings.TrimSpace(text)
}
