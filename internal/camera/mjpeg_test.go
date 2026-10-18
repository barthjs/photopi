package camera

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamMJPEGFrames(t *testing.T) {
	t.Parallel()

	t.Run("writes every frame", func(t *testing.T) {
		t.Parallel()

		first, second := jpeg("first"), jpeg("second")
		// Leading garbage and bytes between frames are skipped, as when a stream starts mid-frame
		src := io.NopCloser(bytes.NewReader(append(append(append([]byte("garbage"), first...), "junk"...), second...)))
		recorder := httptest.NewRecorder()

		err := streamMJPEGFrames(t.Context(), src, recorder, "frame")

		require.ErrorIs(t, err, io.EOF, "the stream ends when the source does")
		assert.Equal(t, part("frame", first)+part("frame", second), recorder.Body.String())
		assert.True(t, recorder.Flushed, "every part is flushed to the client")
	})

	t.Run("keeps marker bytes inside frames", func(t *testing.T) {
		t.Parallel()

		// A lone 0xFF or an SOI sequence inside the image data does not end the frame
		frame := jpeg("a\xff\x00b\xff\xd8c")
		var out bytes.Buffer

		err := streamMJPEGFrames(t.Context(), io.NopCloser(bytes.NewReader(frame)), &out, "b")

		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, part("b", frame), out.String())
	})

	t.Run("reads frames split across reads", func(t *testing.T) {
		t.Parallel()

		first, second := jpeg("a\xffb"), jpeg("\xd9c")
		// One byte per read splits every marker between two reads
		src := io.NopCloser(iotest.OneByteReader(bytes.NewReader(append(append([]byte("x\xff"), first...), second...))))
		var out bytes.Buffer

		err := streamMJPEGFrames(t.Context(), src, &out, "b")

		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, part("b", first)+part("b", second), out.String())
	})

	t.Run("reads frames larger than the read buffer", func(t *testing.T) {
		t.Parallel()

		frame := jpeg(strings.Repeat("\xd9data", readBufferBytes/2))
		var out bytes.Buffer

		err := streamMJPEGFrames(t.Context(), io.NopCloser(bytes.NewReader(frame)), &out, "b")

		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, part("b", frame), out.String())
	})

	t.Run("drops oversized frames", func(t *testing.T) {
		t.Parallel()

		oversized := jpeg(strings.Repeat("x", maxFrameBytes))
		next := jpeg("next")
		var out bytes.Buffer

		err := streamMJPEGFrames(t.Context(), io.NopCloser(bytes.NewReader(append(oversized, next...))), &out, "b")

		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, part("b", next), out.String(), "the reader skips the oversized frame and resynchronizes on the next one")
	})

	t.Run("drops an incomplete frame", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		err := streamMJPEGFrames(t.Context(), io.NopCloser(strings.NewReader("\xff\xd8no end")), &out, "b")

		require.ErrorIs(t, err, io.EOF)
		assert.Empty(t, out.String())
	})

	t.Run("stops on a canceled context before reading", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		var out bytes.Buffer

		err := streamMJPEGFrames(ctx, io.NopCloser(bytes.NewReader(jpeg("x"))), &out, "b")

		require.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, out.String())
	})

	t.Run("prefers the context over a read error after cancel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		src := &cancelingReader{
			cancel: cancel,
		}

		err := streamMJPEGFrames(ctx, src, io.Discard, "b")

		require.ErrorIs(t, err, context.Canceled, "the context error wins over the read error it caused")
	})

	frame := jpeg("data")
	header := fmt.Sprintf("--b\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame))
	writeTests := []struct {
		name    string
		allowed int
	}{
		{
			name:    "reports a failing write of the header",
			allowed: 0,
		},
		{
			name:    "reports a failing write of the frame",
			allowed: len(header),
		},
		{
			name:    "reports a failing write of the trailer",
			allowed: len(header) + len(frame),
		},
	}

	for _, tt := range writeTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &failingWriter{
				allowed: tt.allowed,
			}

			err := streamMJPEGFrames(t.Context(), io.NopCloser(bytes.NewReader(frame)), w, "b")

			require.ErrorIs(t, err, errWriteFailed)
			assert.True(t, isIntentionalStop(err), "a failing client does not count as a camera failure")
		})
	}

	t.Run("prefers the context over a write error after cancel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		w := &cancelingWriter{
			cancel: cancel,
		}

		err := streamMJPEGFrames(ctx, io.NopCloser(bytes.NewReader(jpeg("data"))), w, "b")

		require.ErrorIs(t, err, context.Canceled)
	})
}

// part is how streamMJPEGFrames writes one frame.
func part(boundary string, frame []byte) string {
	return fmt.Sprintf("--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n%s\r\n", boundary, len(frame), frame)
}

// cancelingReader cancels the context and then fails, as a killed source process does.
type cancelingReader struct {
	cancel context.CancelFunc
}

// Read cancels the context and fails.
func (r *cancelingReader) Read([]byte) (int, error) {
	r.cancel()
	return 0, io.ErrUnexpectedEOF
}

// Close does nothing.
func (r *cancelingReader) Close() error {
	return nil
}

// errWriteFailed is the error of a failingWriter.
var errWriteFailed = errors.New("write failed")

// failingWriter accepts allowed bytes and fails every write after that.
type failingWriter struct {
	allowed int
	written int
}

// Write accepts p while it fits into the allowed bytes and fails otherwise.
func (w *failingWriter) Write(p []byte) (int, error) {
	if w.written+len(p) > w.allowed {
		return 0, errWriteFailed
	}
	w.written += len(p)
	return len(p), nil
}

// cancelingWriter cancels the context and then fails, as a write to a disconnected client does.
type cancelingWriter struct {
	cancel context.CancelFunc
}

// Write cancels the context and fails.
func (w *cancelingWriter) Write([]byte) (int, error) {
	w.cancel()
	return 0, errWriteFailed
}

func TestStreamFrames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		frames int
	}{
		{
			name:   "rejects a stream without a first frame",
			frames: 0,
		},
		{
			name:   "rejects a stream whose frames stop",
			frames: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := newStallingReader(bytes.Repeat(jpeg("frame"), tt.frames))
			var out bytes.Buffer

			err := streamFrames(t.Context(), src, &out, "b", frameTimeouts{
				first: 50 * time.Millisecond,
				next:  50 * time.Millisecond,
			})

			require.Error(t, err)
			assert.ErrorContains(t, err, "no preview frame")
			assert.False(t, isIntentionalStop(err), "a stalled camera counts as a failure")
			assert.Equal(t, tt.frames, strings.Count(out.String(), "--b\r\n"))
		})
	}
}

// jpeg builds a minimal byte sequence framed like a JPEG image, with body between the markers.
func jpeg(body string) []byte {
	return append(append(append([]byte{}, jpegSOI...), body...), jpegEOI...)
}

// stallingReader returns its data and then blocks until it is closed, as a hanging camera tool does.
type stallingReader struct {
	data   *bytes.Reader
	closed chan struct{}
}

// newStallingReader creates a stallingReader that returns data first.
func newStallingReader(data []byte) *stallingReader {
	return &stallingReader{
		data:   bytes.NewReader(data),
		closed: make(chan struct{}),
	}
}

// Read returns the remaining data, then blocks until Close.
func (r *stallingReader) Read(p []byte) (int, error) {
	if r.data.Len() > 0 {
		return r.data.Read(p)
	}
	<-r.closed
	return 0, errors.New("read from closed reader")
}

// Close unblocks a pending Read.
func (r *stallingReader) Close() error {
	close(r.closed)
	return nil
}
