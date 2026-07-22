package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 127.0.0.1:1 refuses instantly, so each attempt fails on the ping rather than
// on a dial timeout.
const unreachableURL = "postgres://user:pass@127.0.0.1:1/bakery?sslmode=disable"

func TestOpenPostgresStopsWhenTheContextIsCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	started := time.Now()
	pool, err := OpenPostgres(ctx, unreachableURL)
	elapsed := time.Since(started)

	if pool != nil {
		pool.Close()
		t.Fatal("got a pool for an unreachable database")
	}
	if err == nil {
		t.Fatal("want an error")
	}
	// Without honouring the context this would keep backing off for ~31s.
	if elapsed > 5*time.Second {
		t.Errorf("waited %s; the retry loop ignored the cancelled context", elapsed)
	}
}

func TestOpenPostgresRequiresAURL(t *testing.T) {
	t.Parallel()
	if _, err := OpenPostgres(context.Background(), ""); err == nil {
		t.Fatal("want an error for an empty url")
	}
}

func TestOpenPostgresRetriesBeforeGivingUp(t *testing.T) {
	// Not parallel: shrinks the shared retry ladder so the test does not sleep
	// through the real 31 seconds.
	restore := shrinkRetryLadder(t, 4, 20*time.Millisecond, 40*time.Millisecond)
	defer restore()

	started := time.Now()
	pool, err := OpenPostgres(context.Background(), unreachableURL)
	elapsed := time.Since(started)

	if pool != nil {
		pool.Close()
		t.Fatal("got a pool for an unreachable database")
	}
	if err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v; want the connection failure", err)
	}
	// 20+40+40ms of backoff between four attempts.
	if elapsed < 100*time.Millisecond {
		t.Errorf("gave up after %s; it did not work through the backoff ladder", elapsed)
	}
}

func shrinkRetryLadder(t *testing.T, attempts int, base, max time.Duration) func() {
	t.Helper()
	oldAttempts, oldBase, oldMax := connectAttempts, connectBackoffBase, connectBackoffMax
	connectAttempts, connectBackoffBase, connectBackoffMax = attempts, base, max
	return func() {
		connectAttempts, connectBackoffBase, connectBackoffMax = oldAttempts, oldBase, oldMax
	}
}
