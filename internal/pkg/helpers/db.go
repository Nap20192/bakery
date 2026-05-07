package helpers

import "github.com/jackc/pgx/v5/pgxpool"

func ClosePool(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	pool.Close()
}
