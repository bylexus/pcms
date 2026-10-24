package webserver

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewResizeSemaphoreSize(t *testing.T) {
	tests := []struct {
		size int
		want int
	}{
		{size: 3, want: 3},
		{size: 1, want: 1},
		{size: 0, want: 1},
		{size: -2, want: 1},
	}
	for _, tt := range tests {
		if got := cap(newResizeSemaphore(tt.size)); got != tt.want {
			t.Errorf("newResizeSemaphore(%d): cap = %d, want %d", tt.size, got, tt.want)
		}
	}
}

func TestResizeSemaphoreAcquireFreeSlot(t *testing.T) {
	s := newResizeSemaphore(2)
	for i := 0; i < 2; i++ {
		if err := s.acquire(context.Background(), time.Second); err != nil {
			t.Fatalf("acquire %d: unexpected error: %v", i, err)
		}
	}
	if len(s) != 2 {
		t.Errorf("used slots = %d, want 2", len(s))
	}
}

func TestResizeSemaphoreAcquireFreeSlotZeroTimeout(t *testing.T) {
	s := newResizeSemaphore(1)
	if err := s.acquire(context.Background(), 0); err != nil {
		t.Fatalf("acquire with free slot and zero timeout: unexpected error: %v", err)
	}
}

func TestResizeSemaphoreAcquireTimeout(t *testing.T) {
	s := newResizeSemaphore(1)
	if err := s.acquire(context.Background(), time.Second); err != nil {
		t.Fatalf("first acquire: unexpected error: %v", err)
	}

	start := time.Now()
	err := s.acquire(context.Background(), 20*time.Millisecond)
	if !errors.Is(err, errResizeSlotTimeout) {
		t.Fatalf("second acquire: err = %v, want errResizeSlotTimeout", err)
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Errorf("second acquire returned after %s, want >= 20ms", elapsed)
	}
	if len(s) != 1 {
		t.Errorf("used slots = %d, want 1", len(s))
	}
}

func TestResizeSemaphoreAcquireContextCancelled(t *testing.T) {
	s := newResizeSemaphore(1)
	if err := s.acquire(context.Background(), time.Second); err != nil {
		t.Fatalf("first acquire: unexpected error: %v", err)
	}

	// Already cancelled context:
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.acquire(ctx, time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("acquire with cancelled context: err = %v, want context.Canceled", err)
	}

	// Context cancelled while waiting:
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	if err := s.acquire(ctx, time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("acquire cancelled while waiting: err = %v, want context.Canceled", err)
	}
	if len(s) != 1 {
		t.Errorf("used slots = %d, want 1", len(s))
	}
}

func TestResizeSemaphoreRelease(t *testing.T) {
	s := newResizeSemaphore(1)
	if err := s.acquire(context.Background(), time.Second); err != nil {
		t.Fatalf("first acquire: unexpected error: %v", err)
	}
	s.release()
	if len(s) != 0 {
		t.Fatalf("used slots after release = %d, want 0", len(s))
	}
	if err := s.acquire(context.Background(), 20*time.Millisecond); err != nil {
		t.Fatalf("acquire after release: unexpected error: %v", err)
	}
}

func TestResizeSemaphoreReleaseWakesWaiter(t *testing.T) {
	s := newResizeSemaphore(1)
	if err := s.acquire(context.Background(), time.Second); err != nil {
		t.Fatalf("first acquire: unexpected error: %v", err)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		s.release()
	}()
	if err := s.acquire(context.Background(), 5*time.Second); err != nil {
		t.Fatalf("waiting acquire: unexpected error: %v", err)
	}
}
