package core

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// RequestTracker tracks active proxy requests for graceful shutdown
type RequestTracker struct {
	activeCount int64
	wg          sync.WaitGroup
	shutdownCh  chan struct{}
	isShutdown  atomic.Bool
}

// NewRequestTracker creates a new request tracker
func NewRequestTracker() *RequestTracker {
	return &RequestTracker{
		shutdownCh: make(chan struct{}),
	}
}

// Add increments the active request count
// Returns false if shutdown is in progress (request should be rejected)
func (t *RequestTracker) Add() bool {
	if t.isShutdown.Load() {
		return false
	}
	t.wg.Add(1)
	atomic.AddInt64(&t.activeCount, 1)
	return true
}

// Done decrements the active request count
func (t *RequestTracker) Done() {
	atomic.AddInt64(&t.activeCount, -1)
	t.wg.Done()
}

// ActiveCount returns the current number of active requests
func (t *RequestTracker) ActiveCount() int64 {
	return atomic.LoadInt64(&t.activeCount)
}

// WaitWithTimeout waits for all active requests to complete with a timeout
// Returns true if all requests completed, false if timeout occurred
func (t *RequestTracker) WaitWithTimeout(timeout time.Duration) bool {
	t.isShutdown.Store(true)
	close(t.shutdownCh)

	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// WaitWithContext waits for all active requests to complete or context cancellation
// Returns true if all requests completed, false if context was cancelled
func (t *RequestTracker) WaitWithContext(ctx context.Context) bool {
	t.isShutdown.Store(true)
	close(t.shutdownCh)

	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

// IsShuttingDown returns true if shutdown has been initiated
func (t *RequestTracker) IsShuttingDown() bool {
	return t.isShutdown.Load()
}

// ShutdownCh returns a channel that is closed when shutdown begins
func (t *RequestTracker) ShutdownCh() <-chan struct{} {
	return t.shutdownCh
}

// GracefulShutdown initiates graceful shutdown and waits for requests to complete
// maxWait: maximum time to wait for requests to complete
// checkInterval: how often to log remaining requests
func (t *RequestTracker) GracefulShutdown(maxWait time.Duration, checkInterval time.Duration) bool {
	t.isShutdown.Store(true)
	close(t.shutdownCh)

	activeCount := t.ActiveCount()
	if activeCount == 0 {
		log.Printf("[RequestTracker] No active requests, shutdown immediate")
		return true
	}

	log.Printf("[RequestTracker] Graceful shutdown initiated, waiting for %d active requests", activeCount)

	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	deadline := time.After(maxWait)

	for {
		select {
		case <-done:
			log.Printf("[RequestTracker] All requests completed, shutdown clean")
			return true
		case <-ticker.C:
			remaining := t.ActiveCount()
			log.Printf("[RequestTracker] Waiting for %d active requests to complete...", remaining)
		case <-deadline:
			remaining := t.ActiveCount()
			log.Printf("[RequestTracker] Timeout reached, %d requests still active, forcing shutdown", remaining)
			return false
		}
	}
}
