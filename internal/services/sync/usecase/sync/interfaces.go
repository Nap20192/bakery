// Package syncuc is the application layer of the iiko sync service: it fetches
// the iiko snapshot and persists it. Fetching is behind the IikoClient port,
// persistence behind the Repository port.
package syncuc

import (
	"context"

	"bakery/internal/outbound/iiko"
)

type UseCase interface {
	Run(ctx context.Context) error
	SyncOnce(ctx context.Context) error
}

// IikoClient is the fetch port (satisfied by *iiko.Client).
type IikoClient interface {
	Auth() error
	Logout() error
	ListProductsWithCategories() (*iiko.NomenclatureResponse, error)
	AssemblyChartsGetAll(dateFrom, dateTo string, includeDeleted, includePrepared bool) (*iiko.ChartResultDto, error)
}

// Repository persists a fetched snapshot (sync-run lifecycle + transactional
// upserts).
type Repository interface {
	SaveSnapshot(ctx context.Context, catalog *iiko.NomenclatureResponse, charts *iiko.ChartResultDto, syncDate string) error
}
