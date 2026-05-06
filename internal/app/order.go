package app

import (
	"context"

	"bakery/internal/app/spec"
	"bakery/internal/domain"
	"bakery/internal/repo/sqlc"
)

type OrderService struct {
	queries   *sqlc.Queries
	orderSpec *spec.OrderSpec
}

func NewOrderService(queries *sqlc.Queries) *OrderService {
	return &OrderService{
		queries:   queries,
		orderSpec: spec.NewOrderSpec(queries),
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input domain.CreateOrderInput) error {
	return nil
}

type BulkOrderValidationResult = spec.BulkOrderValidationResult
type BulkOrderValidationItem = spec.BulkOrderValidationItem

func (s *OrderService) ValidateBulkOrder(ctx context.Context, text string) (BulkOrderValidationResult, error) {
	return s.orderSpec.ValidateBulkOrder(ctx, []byte(text))
}
