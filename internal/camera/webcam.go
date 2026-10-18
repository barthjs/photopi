package camera

import (
	"context"
	"io"
)

// webcamCamera is the backend counterpart of the browser-webcam source.
// The browser renders its own preview and takes its own stills via getUserMedia.
type webcamCamera struct{}

// newWebcamCamera creates a new webcamCamera instance.
func newWebcamCamera() *webcamCamera {
	return &webcamCamera{}
}

// StreamPreview fails with ErrUnsupported, since the browser shows the webcam preview itself.
func (webcamCamera) StreamPreview(_ context.Context, _ io.Writer, _ string) error {
	return ErrUnsupported
}

// Capture fails with ErrUnsupported, since the browser takes webcam pictures itself.
func (webcamCamera) Capture(_ context.Context) (*[]byte, error) {
	return nil, ErrUnsupported
}
