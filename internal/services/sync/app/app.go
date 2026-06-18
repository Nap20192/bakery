// Package syncapp is the composition root of the iiko sync service.
package syncapp

import (
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/outbound/iiko"
	syncrepo "bakery/internal/services/sync/infra/repo"
	syncuc "bakery/internal/services/sync/usecase/sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(client *iiko.Client, db *pgxpool.Pool, queries *sqlc.Queries, interval time.Duration) syncuc.UseCase {
	return syncuc.NewService(client, syncrepo.New(db, queries), interval)
}
