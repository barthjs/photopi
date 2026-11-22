package camera

import (
	"context"
	"sync"
)

// deviceSession gives a capture or a new preview exclusive access to a camera device once a running preview
// has stopped and its process released the device, or until the context of the caller ends.
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

// stopAndWait stops a running preview, waits until it released the device, and keeps the device owned until release,
// so a capture and a concurrent StreamPreview never both own it.
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
