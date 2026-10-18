package camera

import (
	"context"
	"sync"
)

// deviceSession coordinates exclusive access to a camera device that only tolerates one running process at a time
// (rpicam-vid/gphoto2's live view both hold it exclusively): a running preview must be fully stopped,
// and its process confirmed to have released the device, before a capture or a new preview claims it.
// rpiCam and gphotoCam each wrap one of these around their own preview/capture commands.
//
// Every wait honors the caller's context, so a preview process that does not exit cannot block a capture
// or a new preview forever.
type deviceSession struct {
	// owner is a one-slot semaphore, held while a caller stops the preview and, for a capture, until the still is taken.
	// A channel instead of a mutex lets the wait for it end with the context.
	owner chan struct{}

	// mu guards cancel and done and is never held while waiting.
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// newDeviceSession creates a deviceSession without a running preview.
func newDeviceSession() *deviceSession {
	return &deviceSession{
		owner: make(chan struct{}, 1),
	}
}

// startPreview stops and waits for any previous preview to fully release the device,
// then registers a new one. It returns the context to run the preview process under
// and a finish func the caller must call once that process has actually exited (after Wait returns),
// so a concurrent stopAndWait can't return before the device is really free.
func (d *deviceSession) startPreview(ctx context.Context) (context.Context, func(), error) {
	if err := d.acquire(ctx); err != nil {
		return nil, nil, err
	}
	defer d.release()

	if err := d.stopPreview(ctx); err != nil {
		return nil, nil, err
	}

	innerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	d.mu.Lock()
	d.cancel = cancel
	d.done = done
	d.mu.Unlock()

	return innerCtx, func() {
		cancel()
		close(done)

		d.mu.Lock()
		if d.done == done {
			d.cancel = nil
			d.done = nil
		}
		d.mu.Unlock()
	}, nil
}

// stopAndWait stops any currently running preview and waits for it to fully release the device,
// then keeps the device owned until release is called, so a capture and a concurrent StreamPreview
// can never both think they own the device at once.
func (d *deviceSession) stopAndWait(ctx context.Context) (func(), error) {
	if err := d.acquire(ctx); err != nil {
		return nil, err
	}

	if err := d.stopPreview(ctx); err != nil {
		d.release()
		return nil, err
	}

	return d.release, nil
}

// acquire waits until the caller owns the device or ctx ends.
func (d *deviceSession) acquire(ctx context.Context) error {
	// Checked first, since select picks randomly when the slot is free as well
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case d.owner <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// stopPreview cancels the currently running preview, if any, and waits until its finish func ran or ctx ends.
// Callers must own the device.
func (d *deviceSession) stopPreview(ctx context.Context) error {
	d.mu.Lock()
	cancel, done := d.cancel, d.done
	d.mu.Unlock()

	if cancel == nil {
		return nil
	}

	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// release gives up the ownership taken by acquire.
func (d *deviceSession) release() {
	<-d.owner
}
