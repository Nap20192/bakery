// Package rabbitmq provides the AMQP connection and the publish/consume
// adapters used as the project's event bus.
package rabbitmq

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	retryTimes     = 5
	backOffSeconds = 2
)

type ConnString string

var ErrCannotConnect = errors.New("cannot connect to rabbitmq")

// Conn keeps a usable AMQP connection.
//
// amqp091-go never reconnects on its own. Once the broker drops the connection
// — a restart, a failover, an idle reap — every Channel() call on that
// *amqp.Connection fails with 504 "channel/connection is not open" for the rest
// of the process lifetime. Holding a single connection from startup therefore
// turns a transient outage into a permanent one: the outbox relay kept retrying
// every two seconds against a dead connection and never drained.
//
// Conn hands out channels rather than the raw connection, so the redial lives
// in one place. Safe for concurrent use.
type Conn struct {
	url ConnString

	mu   sync.Mutex
	conn *amqp.Connection
}

// NewConn dials RabbitMQ with a small bounded retry/backoff. It dials eagerly
// so a misconfigured URL fails at startup rather than on the first publish.
func NewConn(url ConnString) (*Conn, error) {
	client := &Conn{url: url}
	if _, err := client.live(); err != nil {
		return nil, err
	}
	return client, nil
}

// Channel opens a channel, redialing first when the cached connection is gone.
func (c *Conn) Channel() (*amqp.Channel, error) {
	conn, err := c.live()
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err == nil {
		return channel, nil
	}
	// The connection can die between the liveness check and this call, so one
	// failure is not yet evidence that the broker is unreachable.
	c.drop(conn)
	conn, err = c.live()
	if err != nil {
		return nil, err
	}
	return conn.Channel()
}

// Close releases the current connection. Later calls redial.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	if err != nil {
		return fmt.Errorf("close rabbitmq connection: %w", err)
	}
	return nil
}

func (c *Conn) live() (*amqp.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil && !c.conn.IsClosed() {
		return c.conn, nil
	}
	c.conn = nil

	var attempts int
	for {
		conn, err := amqp.Dial(string(c.url))
		if err == nil {
			slog.Info("connected to rabbitmq")
			c.conn = conn
			return conn, nil
		}
		attempts++
		slog.Error("failed to connect to rabbitmq", "error", err, "attempt", attempts)
		if attempts > retryTimes {
			// Callers retry on their own schedule — the relay ticks, the
			// consumer restarts — so give up here instead of blocking them.
			return nil, errors.Join(ErrCannotConnect, err)
		}
		time.Sleep(backOffSeconds * time.Second)
	}
}

// drop discards conn only if it is still the cached one, so a redial performed
// by another goroutine is not thrown away.
func (c *Conn) drop(conn *amqp.Connection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != conn {
		return
	}
	_ = c.conn.Close()
	c.conn = nil
}
