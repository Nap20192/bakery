package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	orderdomain "bakery/internal/domain/order"
	"bakery/internal/outbound/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
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
	assertContains(t, messages, "целое число")
	assertContains(t, messages, "Код продукта не найден")
	assertContains(t, messages, "Строка не распознана")
	assertContains(t, messages, `Позиция с кодом 15635 повторяется`)
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
	assertContains(t, messages, "Не удалось проверить код продукта")
}

func TestOrderServiceValidateBulkOrderResolvesDishNames(t *testing.T) {
	queries := &fakeOrderQueries{
		dishExistsByCode: map[string]int64{"15647": 1},
		dishCatalogByName: map[string][]sqlc.DishCatalog{
			"сосиска в тесте": {{Code: "15647", Name: "Сосиска в тесте", Theme: "ПИРОЖКИ И САМСА"}},
		},
	}
	svc := NewOrderService(queries)

	result := svc.ValidateBulkOrder(context.Background(), "Сосиска в тесте 5")

	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
	if len(result.ValidItems) != 1 {
		t.Fatalf("valid items = %d, want 1", len(result.ValidItems))
	}
	item := result.ValidItems[0]
	if item.Code != "15647" || item.ProductName != "Сосиска в тесте" || item.Quantity != 5 {
		t.Fatalf("resolved item = %#v", item)
	}
}

func TestOrderServiceValidateBulkOrderReportsUnknownDishName(t *testing.T) {
	queries := &fakeOrderQueries{dishCatalogByName: map[string][]sqlc.DishCatalog{}}
	svc := NewOrderService(queries)

	result := svc.ValidateBulkOrder(context.Background(), "Неизвестное блюдо 2")

	if len(result.ValidItems) != 0 {
		t.Fatalf("valid items = %d, want 0", len(result.ValidItems))
	}
	messages := validationMessages(result.Errors)
	assertContains(t, messages, "не найдено в справочнике")
}

func TestOrderServiceValidateBulkOrderReportsAmbiguousDishName(t *testing.T) {
	queries := &fakeOrderQueries{
		dishCatalogByName: map[string][]sqlc.DishCatalog{
			"булочка": {
				{Code: "1", Name: "Булочка", Theme: "A"},
				{Code: "2", Name: "Булочка", Theme: "B"},
			},
		},
	}
	svc := NewOrderService(queries)

	result := svc.ValidateBulkOrder(context.Background(), "Булочка 2")

	if len(result.ValidItems) != 0 {
		t.Fatalf("valid items = %d, want 0", len(result.ValidItems))
	}
	messages := validationMessages(result.Errors)
	assertContains(t, messages, "найдено несколько раз")
}

func TestOrderServiceListOrderTemplatesBuildsFromDishCatalog(t *testing.T) {
	queries := &fakeOrderQueries{
		dishCatalogItems: []sqlc.DishCatalog{
			{Code: "15542", Name: "Кокрок с картофелем", Theme: "КОКРОКИ"},
			{Code: "15544", Name: "Кокрок с творогом", Theme: "КОКРОКИ"},
			{Code: "15646", Name: "Самса с курицей", Theme: "САМСА И УЧПУЧМАК"},
		},
	}
	svc := NewOrderService(queries)

	templates, err := svc.ListOrderTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListOrderTemplates returned error: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("templates = %d, want 2: %#v", len(templates), templates)
	}
	if templates[0].Body != "КОКРОКИ\nКокрок с картофелем 0\nКокрок с творогом 0" {
		t.Fatalf("first template body = %q", templates[0].Body)
	}
	if strings.Contains(templates[0].Body, "15542") || strings.Contains(templates[1].Body, "15646") {
		t.Fatalf("template bodies must not contain dish codes: %#v", templates)
	}

	combined, err := svc.CombinedOrderTemplate(context.Background())
	if err != nil {
		t.Fatalf("CombinedOrderTemplate returned error: %v", err)
	}
	assertContains(t, combined, "КОКРОКИ\nКокрок с картофелем 0")
	assertContains(t, combined, "САМСА И УЧПУЧМАК\nСамса с курицей 0")
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
	if !queries.deleteOrdersCreatedBefore.Valid {
		t.Fatal("cutoff is not valid")
	}
	if !queries.deleteOrdersCreatedBefore.Time.Equal(time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("cutoff = %v", queries.deleteOrdersCreatedBefore)
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
	dishCatalogByName         map[string][]sqlc.DishCatalog
	dishCatalogItems          []sqlc.DishCatalog
	dishCatalogErrorsByName   map[string]error
	deleteOrdersCreatedBefore pgtype.Timestamptz
}

func (q *fakeOrderQueries) DishExistsByCode(_ context.Context, code string) (int64, error) {
	if err := q.dishErrorsByCode[code]; err != nil {
		return 0, err
	}
	return q.dishExistsByCode[code], nil
}

func (q *fakeOrderQueries) ListDishCatalogItems(_ context.Context) ([]sqlc.DishCatalog, error) {
	result := make([]sqlc.DishCatalog, len(q.dishCatalogItems))
	copy(result, q.dishCatalogItems)
	return result, nil
}

func (q *fakeOrderQueries) ListDishCatalogItemsByName(_ context.Context, name string) ([]sqlc.DishCatalog, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if err := q.dishCatalogErrorsByName[key]; err != nil {
		return nil, err
	}
	items := q.dishCatalogByName[key]
	result := make([]sqlc.DishCatalog, len(items))
	copy(result, items)
	return result, nil
}

func (q *fakeOrderQueries) DeleteOrdersCreatedBefore(_ context.Context, createdAtBefore pgtype.Timestamptz) (int64, error) {
	q.deleteOrdersCreatedBefore = createdAtBefore
	return 7, nil
}
