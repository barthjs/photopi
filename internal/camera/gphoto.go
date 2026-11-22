package camera

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/rs/zerolog"
)

// gphotoCam drives a tethered camera through gphoto2 over USB, whose settings are all made on the camera body,
// and stops a running live view for a still, since only one gphoto2 process can hold the connection.
type gphotoCam struct {
	logger zerolog.Logger

	device *deviceSession
}

// newGphotoCam creates a new gphotoCam instance and verifies gphoto2 is on PATH.
func newGphotoCam(logger zerolog.Logger) (*gphotoCam, error) {
	if _, err := exec.LookPath("gphoto2"); err != nil {
		return nil, fmt.Errorf("camera: required tool \"gphoto2\" not found in PATH: %w", err)
	}

	return &gphotoCam{
		logger: logger,
		device: newDeviceSession(),
	}, nil
}

// StreamPreview starts gphoto2's live view movie capture and forwards its raw MJPEG output to w
// as a multipart/x-mixed-replace stream, until ctx is canceled, the client disconnects,
// or a concurrent Capture stops it to take the shutter over the same USB connection.
// gphoto2 handles SIGTERM like SIGINT and closes its PTP session before it exits.
func (c *gphotoCam) StreamPreview(ctx context.Context, w io.Writer, boundary string) error {
	return streamPreviewProcess(ctx, c.logger, c.device, command{
		label: "gphoto2 live view",
		name:  "gphoto2",
		args: []string{
			"--capture-movie",
			"--stdout",
		},
	}, w, boundary)
}

// Capture takes a still at the settings of the camera body, after stopping a running live view.
func (c *gphotoCam) Capture(ctx context.Context) (*[]byte, error) {
	// The live view is stopped with SIGTERM and a grace period, so gphoto2 closes its PTP session.
	// A live view killed with SIGKILL leaves the PTP state of the camera unsettled,
	// and the next still can come back truncated, a JPEG that looks valid but is incomplete.
	return captureProcess(ctx, c.logger, c.device, command{
		label: "gphoto2 capture",
		name:  "gphoto2",
		args: []string{
			"--capture-image-and-download",
			"--stdout",
		},
	})
}
