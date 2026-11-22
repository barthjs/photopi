package camera

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// jpegSOI and jpegEOI mark the start and end of a JPEG image, the only framing of the MJPEG output of the camera tools.
// A JPEG encoder byte-stuffs a 0x00 after every literal 0xFF in the image data, so EOI occurs only at the end.
var (
	jpegSOI = []byte{0xFF, 0xD8}
	jpegEOI = []byte{0xFF, 0xD9}
)

const (
	// firstFrameTimeout bounds the wait for the first frame, which includes starting the camera.
	firstFrameTimeout = 10 * time.Second
	// frameTimeout bounds the wait for each further frame, so a stalled tool ends the stream.
	frameTimeout = 5 * time.Second
	// maxFrameBytes bounds a single frame, so a stream without end markers cannot grow the buffer forever.
	maxFrameBytes = 8 << 20
	// readBufferBytes is the size of the read buffer the frames are searched in.
	readBufferBytes = 256 << 10
)

// errFrameTooLarge reports a frame that exceeded maxFrameBytes and was dropped.
var errFrameTooLarge = errors.New("camera: preview frame too large")

// previewWriteError wraps a failure to write the preview to the client, as opposed to reading it from the camera.
type previewWriteError struct {
	err error
}

// Error returns the message of the wrapped error.
func (e *previewWriteError) Error() string {
	return "camera: writing preview: " + e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *previewWriteError) Unwrap() error {
	return e.err
}

// frameTimeouts bound the waits for the frames of a stream.
type frameTimeouts struct {
	first time.Duration
	next  time.Duration
}

// streamMJPEGFrames reads a raw MJPEG byte stream from src and writes each JPEG frame it finds to w
// as one part of a multipart/x-mixed-replace body with the given boundary,
// flushing after each part so the browser renders it immediately instead of waiting for more buffered data.
// It runs until ctx is canceled, reading src or writing w fails, or no frame arrives in time.
func streamMJPEGFrames(ctx context.Context, src io.ReadCloser, w io.Writer, boundary string) error {
	return streamFrames(ctx, src, w, boundary, frameTimeouts{
		first: firstFrameTimeout,
		next:  frameTimeout,
	})
}

// streamFrames implements streamMJPEGFrames with the given timeouts. A frame that does not arrive in time closes src,
// which unblocks the pending read, and ends the stream with an error.
func streamFrames(ctx context.Context, src io.ReadCloser, w io.Writer, boundary string, timeouts frameTimeouts) error {
	flusher, _ := w.(http.Flusher)
	frames := newFrameReader(src)

	var stalled atomic.Bool
	watchdog := time.AfterFunc(timeouts.first, func() {
		stalled.Store(true)
		_ = src.Close()
	})
	defer watchdog.Stop()
	wait := timeouts.first

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		frame, err := frames.next()
		if err != nil {
			// A canceled context stops the source process, and the blocking read fails with a plain I/O error,
			// so the context error tells an intentional stop apart from a stream failure.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if stalled.Load() {
				return fmt.Errorf("camera: no preview frame within %s", wait)
			}
			return err
		}

		// Writing to a slow client does not count against the camera
		watchdog.Stop()
		if err := writePart(w, boundary, frame); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return &previewWriteError{
				err: err,
			}
		}
		if flusher != nil {
			flusher.Flush()
		}
		wait = timeouts.next
		watchdog.Reset(wait)
	}
}

// writePart writes one frame as a part of the multipart body.
func writePart(w io.Writer, boundary string, frame []byte) error {
	if _, err := fmt.Fprintf(w, "--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", boundary, len(frame)); err != nil {
		return err
	}
	if _, err := w.Write(frame); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\r\n")
	return err
}

// frameReader splits a raw MJPEG byte stream into its JPEG frames.
// It reuses one frame buffer, so a frame is only valid until the next call of next.
type frameReader struct {
	r     *bufio.Reader
	frame []byte
}

// newFrameReader creates a frameReader reading from src.
func newFrameReader(src io.Reader) *frameReader {
	return &frameReader{
		r: bufio.NewReaderSize(src, readBufferBytes),
	}
}

// next reads and returns the next complete JPEG image. Any bytes preceding the next start-of-image marker are discarded,
// and so is a frame larger than maxFrameBytes, so the reader resynchronizes on its own
// if the stream starts or ever ends up mid-frame.
func (f *frameReader) next() ([]byte, error) {
	for {
		if err := f.skipToStart(); err != nil {
			return nil, err
		}

		frame, err := f.readFrame()
		if errors.Is(err, errFrameTooLarge) {
			continue
		}
		return frame, err
	}
}

// skipToStart discards the bytes before the next start-of-image marker.
func (f *frameReader) skipToStart() error {
	for {
		// Fills the buffer if fewer bytes than a marker are buffered
		if _, err := f.r.Peek(len(jpegSOI)); err != nil {
			return err
		}

		buffered, _ := f.r.Peek(f.r.Buffered())
		if i := bytes.Index(buffered, jpegSOI); i >= 0 {
			_, _ = f.r.Discard(i)
			return nil
		}

		// The last byte may be the first half of a marker split across two reads
		_, _ = f.r.Discard(len(buffered) - (len(jpegSOI) - 1))
	}
}

// readFrame reads from the start-of-image marker at the current position up to and including the end-of-image marker.
func (f *frameReader) readFrame() ([]byte, error) {
	f.frame = f.frame[:0]

	for {
		chunk, err := f.r.ReadSlice(jpegEOI[1])
		if len(f.frame)+len(chunk) > maxFrameBytes {
			return nil, errFrameTooLarge
		}
		f.frame = append(f.frame, chunk...)

		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if err != nil {
			return nil, err
		}

		// The second marker byte also occurs on its own, only after 0xFF does it end the image
		n := len(f.frame)
		if n >= len(jpegSOI)+len(jpegEOI) && f.frame[n-2] == jpegEOI[0] {
			return f.frame, nil
		}
	}
}
