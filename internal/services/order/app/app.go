package orderapp

import (
	"database/sql"
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	orderoutbox "bakery/internal/services/order/infra/outbox"
	orderrepo "bakery/internal/services/order/infra/repo"
	orderuc "bakery/internal/services/order/usecase/order"
)

func New(queries *sqlc.Queries, db *sql.DB) orderuc.UseCase {
	return orderuc.NewService(orderrepo.New(queries, db))
}

func NewOutboxRelay(queries *sqlc.Queries, handler orderoutbox.Handler, interval time.Duration) *orderoutbox.Relay {
	return orderoutbox.NewRelay(queries, handler, interval)
}
