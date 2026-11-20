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

func NewOutboxRelay(queries *sqlc.Queries, handler orderoutbox.Handler, interval time.Duration) *orderoutbox.Relay {
	return orderoutbox.NewRelay(queries, handler, interval)
}
