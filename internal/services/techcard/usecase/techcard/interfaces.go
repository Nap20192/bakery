// Package techcarduc is the application layer of the techcard service.
package techcarduc

import (
	"context"
	"time"

	techcarddomain "bakery/internal/services/techcard/domain"
)

type UseCase interface {
	GetByCode(ctx context.Context, code string, date time.Time) (techcarddomain.TechCard, error)
}

// Repository builds a tech card from the iiko snapshot for a product code/date.
type Repository interface {
	GetByCode(ctx context.Context, code string, date time.Time) (techcarddomain.TechCard, error)
}
