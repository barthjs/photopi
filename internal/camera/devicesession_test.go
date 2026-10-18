package camera

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceSession_startPreview(t *testing.T) {
	t.Parallel()

	t.Run("stops the previous preview", func(t *testing.T) {
		t.Parallel()

		device := newDeviceSession()
		firstCtx, finishFirst, err := device.startPreview(t.Context())
		require.NoError(t, err)
		// The previous preview only finishes once its process released the device
		go func() {
			<-firstCtx.Done()
			finishFirst()
		}()

		secondCtx, finishSecond, err := device.startPreview(t.Context())

		require.NoError(t, err)
		defer finishSecond()
		require.ErrorIs(t, firstCtx.Err(), context.Canceled)
		assert.NoError(t, secondCtx.Err())
	})

	t.Run("keeps the new preview after a late finish of the previous one", func(t *testing.T) {
		t.Parallel()

		device := newDeviceSession()
		firstCtx, finishFirst, err := device.startPreview(t.Context())
		require.NoError(t, err)
		// The old preview finishes only after startPreview registered the new one,
		// since startPreview holds the device while it waits. Its finish must not unregister the new preview.
		finished := make(chan struct{})
		go func() {
			<-firstCtx.Done()
			finishFirst()
			close(finished)
		}()

		secondCtx, finishSecond, err := device.startPreview(t.Context())

		require.NoError(t, err)
		<-finished
		go func() {
			<-secondCtx.Done()
			finishSecond()
		}()
		release, err := device.stopAndWait(t.Context())
		require.NoError(t, err)
		defer release()
		assert.ErrorIs(t, secondCtx.Err(), context.Canceled, "stopAndWait still reached the running preview")
	})

	t.Run("ends the preview with the parent context", func(t *testing.T) {
		t.Parallel()

		device := newDeviceSession()
		parent, cancel := context.WithCancel(t.Context())
		previewCtx, finish, err := device.startPreview(parent)
		require.NoError(t, err)
		defer finish()

		cancel()

		assert.ErrorIs(t, previewCtx.Err(), context.Canceled)
	})

	t.Run("gives up while a capture holds the device", func(t *testing.T) {
		t.Parallel()

		device := newDeviceSession()
		release, err := device.stopAndWait(t.Context())
		require.NoError(t, err)
		defer release()
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		previewCtx, finish, err := device.startPreview(ctx)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Nil(t, previewCtx)
		assert.Nil(t, finish)
	})
}

func TestDeviceSession_stopAndWait(t *testing.T) {
	t.Parallel()

	t.Run("holds the device", func(t *testing.T) {
		t.Parallel()

		device := newDeviceSession()
		previewCtx, finish, err := device.startPreview(t.Context())
		require.NoError(t, err)
		go func() {
			<-previewCtx.Done()
			finish()
		}()

		release, err := device.stopAndWait(t.Context())

		require.NoError(t, err)
		require.ErrorIs(t, previewCtx.Err(), context.Canceled)
		started := make(chan struct{})
		go func() {
			_, finishNext, err := device.startPreview(t.Context())
			assert.NoError(t, err)
			close(started)
			finishNext()
		}()
		select {
		case <-started:
			t.Fatal("a preview started while a capture held the device")
		case <-time.After(50 * time.Millisecond):
		}
		release()
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("the preview did not start after the capture released the device")
		}
	})

	t.Run("returns without a preview", func(t *testing.T) {
		t.Parallel()

		device := newDeviceSession()

		release, err := device.stopAndWait(t.Context())

		require.NoError(t, err)
		release()
	})

	t.Run("gives up on a stuck preview", func(t *testing.T) {
		t.Parallel()

		device := newDeviceSession()
		// The preview never finishes, as a process that ignores its stop signal
		_, finish, err := device.startPreview(t.Context())
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		release, err := device.stopAndWait(ctx)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Nil(t, release)
		// The device is free again for the next attempt once the preview finishes
		finish()
		release, err = device.stopAndWait(t.Context())
		require.NoError(t, err)
		release()
	})

	t.Run("never acquires the device with a canceled context", func(t *testing.T) {
		t.Parallel()

		device := newDeviceSession()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := device.stopAndWait(ctx)

		require.ErrorIs(t, err, context.Canceled)
	})
}
