// Package monitorapp is the composition root of the monitoring service.
package monitorapp

import (
	sqlc "bakery/internal/outbound/db/sqlc"
	monitorrepo "bakery/internal/services/monitor/infra/repo"
	monitoruc "bakery/internal/services/monitor/usecase/monitor"
)

func New(queries *sqlc.Queries) monitoruc.UseCase {
	return monitoruc.NewService(monitorrepo.New(queries))
}
