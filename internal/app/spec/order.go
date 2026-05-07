package spec

import (
	"bakery/internal/domain"
	"bakery/internal/repo/sqlc"
	"context"
	"errors"
	"fmt"
)

type OrderSpec struct {
	queries *sqlc.Queries
}

func NewOrderSpec(queries *sqlc.Queries) *OrderSpec {
	return &OrderSpec{queries: queries}
}

func (s *OrderSpec) ValidateBulkOrderUnique(
	ctx context.Context,
	items []domain.OrderItem,
) (bool, error) {

	existed := make(map[string]bool)
	var errs []error

	for _, item := range items {
		key := fmt.Sprintf("%s_%s", item.Code, item.ProductName)

		if _, exists := existed[key]; exists {
			errs = append(errs,
				fmt.Errorf("duplicate item with code %s and product_name '%s'",
					item.Code, item.ProductName),
			)
			continue
		}

		existed[key] = true
	}

	if len(errs) > 0 {
		return false, errors.Join(errs...)
	}

	return true, nil
}
