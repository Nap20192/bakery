// Package orderapp is the composition root of the order service: it wires the
// use case with its repository, and the outbox relay with its publisher.
package orderapp

import (
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/services/order/eventsourced"
	orderoutbox "bakery/internal/services/order/infra/outbox"
	orderrepo "bakery/internal/services/order/infra/repo"
	orderuc "bakery/internal/services/order/usecase/order"

	"github.com/hallgren/eventsourcing/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Option func(*options)

type options struct {
	eventStore core.EventStore
}

// WithEventStore включает event-sourced запись жизненного цикла заказа:
// команды пишут события в store, репозиторий проецирует их в read model.
// Без опции (bot-бинарь) сервис работает по legacy-пути через репозиторий.
func WithEventStore(store core.EventStore) Option {
	return func(o *options) { o.eventStore = store }
}

func New(queries *sqlc.Queries, db *pgxpool.Pool, opts ...Option) orderuc.UseCase {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	repo := orderrepo.New(queries, db)
	if o.eventStore != nil {
		commands := eventsourced.NewCommands(o.eventStore, repo)
		return orderuc.NewService(repo, orderuc.WithESCommands(commands))
	}
	return orderuc.NewService(repo)
}

// NewOutboxRelay builds the relay that publishes persisted order events.
func NewOutboxRelay(queries *sqlc.Queries, db *pgxpool.Pool, publisher orderoutbox.Publisher, interval time.Duration) *orderoutbox.Relay {
	return orderoutbox.NewRelay(queries, db, publisher, interval)
}
