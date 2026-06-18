// Package orderapp is the composition root of the order service: it wires the
// use case with its repository, and the outbox relay with its publisher.
package orderapp

import (
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	orderoutbox "bakery/internal/services/order/infra/outbox"
	orderrepo "bakery/internal/services/order/infra/repo"
	orderuc "bakery/internal/services/order/usecase/order"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(queries *sqlc.Queries, db *pgxpool.Pool) orderuc.UseCase {
	return orderuc.NewService(orderrepo.New(queries, db))
}

// NewOutboxRelay builds the relay that publishes persisted order events.
func NewOutboxRelay(queries *sqlc.Queries, db *pgxpool.Pool, publisher orderoutbox.Publisher, interval time.Duration) *orderoutbox.Relay {
	return orderoutbox.NewRelay(queries, db, publisher, interval)
}
