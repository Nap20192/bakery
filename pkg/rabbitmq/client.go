// Package rabbitmq provides the AMQP connection and the publish/consume
// adapters used as the project's event bus.
package rabbitmq

import (
	"errors"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	retryTimes     = 5
	backOffSeconds = 2
)

type ConnString string

var ErrCannotConnect = errors.New("cannot connect to rabbitmq")

// NewConn dials RabbitMQ with a small bounded retry/backoff.
func NewConn(url ConnString) (*amqp.Connection, error) {
	var counts int64
	for {
		conn, err := amqp.Dial(string(url))
		if err == nil {
			slog.Info("connected to rabbitmq")
			return conn, nil
		}
		slog.Error("failed to connect to rabbitmq", "error", err)
		counts++
		if counts > retryTimes {
			return nil, ErrCannotConnect
		}
		time.Sleep(backOffSeconds * time.Second)
	}
}
