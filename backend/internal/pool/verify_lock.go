package pool

import (
	"context"
	"sync"
)

// fairLock is a mutual exclusion granted in arrival order, whose waiters give
// up when their context ends.
type fairLock struct {
	mu      sync.Mutex
	held    bool
	waiters []chan struct{}
}

func newFairLock() *fairLock { return &fairLock{} }

// uiLock serializes the UI verification step of the whole backend: ports, the
// database and the browser are resources of the machine, not of a workspace.
var uiLock = newFairLock()

// Acquire blocks until the lock is granted or ctx ends; on cancellation the
// caller leaves the queue and holds nothing.
func (l *fairLock) Acquire(ctx context.Context) error {
	l.mu.Lock()
	if !l.held && len(l.waiters) == 0 {
		l.held = true
		l.mu.Unlock()
		return nil
	}
	ch := make(chan struct{})
	l.waiters = append(l.waiters, ch)
	l.mu.Unlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		l.mu.Lock()
		for i, w := range l.waiters {
			if w == ch {
				l.waiters = append(l.waiters[:i], l.waiters[i+1:]...)
				l.mu.Unlock()
				return ctx.Err()
			}
		}
		l.mu.Unlock()
		// The grant raced the cancellation: it is ours, hand it to the next.
		l.Release()
		return ctx.Err()
	}
}

// Release hands the lock to the oldest waiter, or frees it.
func (l *fairLock) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.waiters) == 0 {
		l.held = false
		return
	}
	next := l.waiters[0]
	l.waiters = l.waiters[1:]
	close(next) // the lock stays held, now by next
}
