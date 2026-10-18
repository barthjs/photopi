package camera

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/rs/zerolog"
)

// gphotoCam drives a tethered DSLR or mirrorless camera through the gphoto2 command-line tool (backed by libgphoto2),
// which talks PTP over USB. Image size, quality, exposure, and autofocus are all controlled on the camera body itself.
//
// gphoto2 only ever lets one of its own processes hold the camera's USB connection at a time:
// a second one fails immediately with "could not claim the USB device". device coordinates that:
// since the live view stream (StreamPreview) runs for as long as a client stays connected,
// a still (Capture) requested while it is running has to actively stop it
// and wait for gphoto2 to actually release the connection first.
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

// StreamPreview starts gphoto2's live view movie capture and forwards its raw MJPEG output
// to w as a multipart/x-mixed-replace stream, until ctx is canceled, the client disconnects,
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

// Capture triggers the shutter and downloads the resulting still straight to stdout,
// at whatever resolution and quality is currently set on the camera body.
// If a live view stream is currently running, it is stopped first,
// and Capture waits for gphoto2 to fully release the USB connection before taking the shot.
//
// The live view is stopped with SIGTERM and a grace period, so gphoto2 closes its PTP session.
// A live view killed with SIGKILL leaves the PTP state of the camera unsettled,
// and the next still can come back truncated, a JPEG that looks valid but is incomplete.
func (c *gphotoCam) Capture(ctx context.Context) (*[]byte, error) {
	return captureProcess(ctx, c.logger, c.device, command{
		label: "gphoto2 capture",
		name:  "gphoto2",
		args: []string{
			"--capture-image-and-download",
			"--stdout",
		},
	})
}
