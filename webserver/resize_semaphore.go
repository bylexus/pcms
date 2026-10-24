package webserver

import (
	"context"
	"errors"
	"time"
)

// errResizeSlotTimeout is returned by resizeSemaphore.acquire when no slot became free
// within the wait timeout.
var errResizeSlotTimeout = errors.New("no free image resize slot within wait timeout")

// resizeSemaphore limits the number of concurrent uncached image renders
// (decode + resize + encode). It is a buffered channel: one element = one used slot.
type resizeSemaphore chan struct{}

// newResizeSemaphore creates a semaphore with size slots (at least 1).
func newResizeSemaphore(size int) resizeSemaphore {
	if size < 1 {
		size = 1
	}
	return make(resizeSemaphore, size)
}

// acquire waits for a free slot. It returns nil when a slot was taken (the caller must
// call release), errResizeSlotTimeout when no slot became free within timeout, or
// ctx.Err() when the context was cancelled (e.g. client disconnect) before.
func (s resizeSemaphore) acquire(ctx context.Context, timeout time.Duration) error {
	// Fast path: take a free slot without starting a timer.
	select {
	case s <- struct{}{}:
		return nil
	default:
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case s <- struct{}{}:
		return nil
	case <-timer.C:
		return errResizeSlotTimeout
	case <-ctx.Done():
		return ctx.Err()
	}
}

// release frees a slot taken by a successful acquire.
func (s resizeSemaphore) release() {
	<-s
}
