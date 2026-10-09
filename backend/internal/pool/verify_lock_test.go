package pool

import (
	"context"
	"testing"
	"time"
)

func waitWaiters(t *testing.T, l *fairLock, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		l.mu.Lock()
		got := len(l.waiters)
		l.mu.Unlock()
		if got == n {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("expected %d waiters", n)
}

func TestFairLockSingleHolder(t *testing.T) {
	l := newFairLock()
	if err := l.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := make(chan struct{})
	go func() {
		_ = l.Acquire(context.Background())
		close(got)
	}()
	waitWaiters(t, l, 1)
	select {
	case <-got:
		t.Fatal("second holder got the lock while it was held")
	case <-time.After(30 * time.Millisecond):
	}
	l.Release()
	select {
	case <-got:
	case <-time.After(time.Second):
		t.Fatal("waiter not granted after release")
	}
	l.Release()
}

func TestFairLockArrivalOrder(t *testing.T) {
	l := newFairLock()
	_ = l.Acquire(context.Background())
	order := make(chan int, 3)
	for i := 1; i <= 3; i++ {
		go func(i int) {
			_ = l.Acquire(context.Background())
			order <- i
		}(i)
		waitWaiters(t, l, i)
	}
	for want := 1; want <= 3; want++ {
		l.Release()
		select {
		case got := <-order:
			if got != want {
				t.Fatalf("expected %d, got %d", want, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("waiter %d not granted", want)
		}
	}
	l.Release()
}

func TestFairLockCancelledWaiterKeepsNext(t *testing.T) {
	l := newFairLock()
	_ = l.Acquire(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() { errc <- l.Acquire(ctx) }()
	waitWaiters(t, l, 1)
	next := make(chan struct{})
	go func() {
		_ = l.Acquire(context.Background())
		close(next)
	}()
	waitWaiters(t, l, 2)

	cancel()
	if err := <-errc; err == nil {
		t.Fatal("cancelled waiter must return an error")
	}
	waitWaiters(t, l, 1)
	l.Release()
	select {
	case <-next:
	case <-time.After(time.Second):
		t.Fatal("next waiter lost after a cancellation")
	}
	l.Release()
	if err := l.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFairLockReleasedOnPanic(t *testing.T) {
	l := newFairLock()
	func() {
		defer func() { _ = recover() }()
		_ = l.Acquire(context.Background())
		defer l.Release()
		panic("boom")
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := l.Acquire(ctx); err != nil {
		t.Fatalf("lock still held after panic: %v", err)
	}
}
