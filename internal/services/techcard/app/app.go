// Package techcardapp is the composition root of the techcard service.
package techcardapp

import (
	sqlc "bakery/internal/outbound/db/sqlc"
	techcardrepo "bakery/internal/services/techcard/infra/repo"
	techcarduc "bakery/internal/services/techcard/usecase/techcard"
)

func New(queries *sqlc.Queries) techcarduc.UseCase {
	return techcarduc.NewService(techcardrepo.New(queries))
}
