package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	orderdomain "bakery/internal/domain/order"
	"bakery/internal/outbound/db/sqlc"
)

func TestOrderServiceValidateBulkOrder(t *testing.T) {
	queries := &fakeOrderQueries{
		dishExistsByCode: map[string]int64{
			"15635": 1,
			"20495": 1,
			"00000": 0,
		},
	}
	svc := NewOrderService(queries)

	result := svc.ValidateBulkOrder(context.Background(), `
15635 Пирожок с капустой 15
20495 Пирожок с картошкой 2.5
00000 Неизвестный продукт 1
bad line
15635 Пирожок с капустой 3
`)

	if len(result.ValidItems) != 3 {
		t.Fatalf("valid items = %d, want 3: %#v", len(result.ValidItems), result.ValidItems)
	}

	messages := validationMessages(result.Errors)
	assertContains(t, messages, "non-negative integers")
	assertContains(t, messages, "product code not found")
	assertContains(t, messages, "invalid format")
	assertContains(t, messages, `duplicate item with code 15635`)
}

func TestOrderServiceValidateBulkOrderReportsDBError(t *testing.T) {
	queries := &fakeOrderQueries{
		dishExistsByCode: map[string]int64{"15635": 1},
		dishErrorsByCode: map[string]error{"20495": errors.New("db unavailable")},
	}
	svc := NewOrderService(queries)

	result := svc.ValidateBulkOrder(context.Background(), `
15635 Пирожок с капустой 15
20495 Пирожок с картошкой 1
`)

	if len(result.ValidItems) != 2 {
		t.Fatalf("valid items = %d, want 2", len(result.ValidItems))
	}
	messages := validationMessages(result.Errors)
	assertContains(t, messages, "failed to validate code: db unavailable")
}

func TestOrderServiceDeleteOrdersOlderThan(t *testing.T) {
	queries := &fakeOrderQueries{}
	svc := NewOrderService(queries)

	deleted, err := svc.DeleteOrdersOlderThan(context.Background(), time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC), 31*24*time.Hour)
	if err != nil {
		t.Fatalf("DeleteOrdersOlderThan returned error: %v", err)
	}
	if deleted != 7 {
		t.Fatalf("deleted = %d, want 7", deleted)
	}
	if !strings.HasPrefix(queries.deleteOrdersCreatedBefore, "2026-04-13T12:00:00") {
		t.Fatalf("cutoff = %q", queries.deleteOrdersCreatedBefore)
	}
}

func validationMessages(errors []orderdomain.BulkOrderValidationError) string {
	var messages []string
	for _, item := range errors {
		messages = append(messages, item.Message)
	}
	return strings.Join(messages, "\n")
}

func assertContains(t *testing.T, value string, needle string) {
	t.Helper()
	if !strings.Contains(value, needle) {
		t.Fatalf("%q does not contain %q", value, needle)
	}
}

type fakeOrderQueries struct {
	sqlc.Querier
	dishExistsByCode          map[string]int64
	dishErrorsByCode          map[string]error
	deleteOrdersCreatedBefore string
}

func (q *fakeOrderQueries) DishExistsByCode(_ context.Context, code string) (int64, error) {
	if err := q.dishErrorsByCode[code]; err != nil {
		return 0, err
	}
	return q.dishExistsByCode[code], nil
}

func (q *fakeOrderQueries) DeleteOrdersCreatedBefore(_ context.Context, createdAtBefore string) (int64, error) {
	q.deleteOrdersCreatedBefore = createdAtBefore
	return 7, nil
}
