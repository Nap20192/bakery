package rabbitmq

import (
	"os"
	"testing"
)

// TestConnRecoversFromADroppedConnection reproduces the outbox incident: the
// broker drops the connection and every later Channel() on the cached
// *amqp.Connection fails with 504 "channel/connection is not open" until the
// process restarts.
//
// Needs a broker: RABBITMQ_TEST_URL=amqp://guest:guest@127.0.0.1:5672/
func TestConnRecoversFromADroppedConnection(t *testing.T) {
	url := os.Getenv("RABBITMQ_TEST_URL")
	if url == "" {
		t.Skip("RABBITMQ_TEST_URL is not set")
	}

	client, err := NewConn(ConnString(url))
	if err != nil {
		t.Fatalf("NewConn: %v", err)
	}
	defer func() { _ = client.Close() }()

	first, err := client.Channel()
	if err != nil {
		t.Fatalf("first Channel: %v", err)
	}
	_ = first.Close()

	// Drop the connection the way the broker would.
	client.mu.Lock()
	dropped := client.conn
	client.mu.Unlock()
	if dropped == nil {
		t.Fatal("no connection was cached")
	}
	if err := dropped.Close(); err != nil {
		t.Fatalf("closing the connection under test: %v", err)
	}
	if !dropped.IsClosed() {
		t.Fatal("connection did not close")
	}

	second, err := client.Channel()
	if err != nil {
		t.Fatalf("Channel after the connection dropped: %v", err)
	}
	defer func() { _ = second.Close() }()

	client.mu.Lock()
	current := client.conn
	client.mu.Unlock()
	if current == dropped {
		t.Error("the dead connection is still cached; Channel() did not redial")
	}
}
