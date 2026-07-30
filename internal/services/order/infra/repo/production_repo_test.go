package orderrepo

import (
	"errors"
	"fmt"
	"testing"

	"bakery/internal/pkg/apperr"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestProductionSheetOrderInsertError(t *testing.T) {
	t.Parallel()

	conflict := productionSheetOrderInsertError(
		fmt.Errorf("insert: %w", &pgconn.PgError{
			Code:           "23505",
			ConstraintName: productionSheetOrderUniqueConstraint,
		}),
		"Г.Х.29.07.26.001",
	)
	appErr := apperr.As(conflict)
	if appErr == nil || appErr.Kind != apperr.KindConflict || appErr.Code != "order.production_exists" {
		t.Fatalf("production conflict = %v, want order.production_exists conflict", conflict)
	}

	cause := &pgconn.PgError{Code: "23505", ConstraintName: "other_unique_key"}
	got := productionSheetOrderInsertError(cause, "Г.Х.29.07.26.001")
	if !errors.Is(got, cause) {
		t.Fatalf("other constraint error = %v, want wrapped original error", got)
	}
}
