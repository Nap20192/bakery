package bot

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryUntilDoneRestartsAfterChannelClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	calls := 0
	done := make(chan struct{})
	go func() {
		defer close(done)
		retryUntilDone(ctx, time.Millisecond, func(ctx context.Context) error {
			calls++
			if calls < 3 {
				return errors.New("rabbitmq channel closed")
			}
			cancel()
			<-ctx.Done()
			return nil
		})
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("retryUntilDone did not stop after ctx cancel")
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (two failures retried, then clean stop)", calls)
	}
}
