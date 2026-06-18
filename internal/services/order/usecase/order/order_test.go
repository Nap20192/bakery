package orderuc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	orderdomain "bakery/internal/services/order/domain"
)

func TestServiceValidateBulkOrder(t *testing.T) {
	repo := &fakeRepo{
		dishExistsByCode: map[string]bool{
			"15635": true,
			"20495": true,
			"00000": false,
		},
	}
	svc := NewService(repo)

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
	assertContains(t, messages, "целое")
	assertContains(t, messages, "Код продукта не найден")
	assertContains(t, messages, "Не распознано")
	assertContains(t, messages, `Позиция с кодом 15635 повторяется`)
}

func TestServiceValidateBulkOrderReportsDBError(t *testing.T) {
	repo := &fakeRepo{
		dishExistsByCode: map[string]bool{"15635": true},
		dishErrorsByCode: map[string]error{"20495": errors.New("db unavailable")},
	}
	svc := NewService(repo)

	result := svc.ValidateBulkOrder(context.Background(), `
15635 Пирожок с капустой 15
20495 Пирожок с картошкой 1
`)

	if len(result.ValidItems) != 2 {
		t.Fatalf("valid items = %d, want 2", len(result.ValidItems))
	}
	assertContains(t, validationMessages(result.Errors), "Не удалось проверить код продукта")
}

func TestServiceValidateBulkOrderResolvesDishNames(t *testing.T) {
	repo := &fakeRepo{
		dishExistsByCode: map[string]bool{"15647": true},
		resolveByName: map[string]DishCatalogItem{
			"сосиска в тесте": {Code: "15647", Name: "Сосиска в тесте", Theme: "ПИРОЖКИ И САМСА"},
		},
	}
	svc := NewService(repo)

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

func TestServiceValidateBulkOrderReportsUnknownDishName(t *testing.T) {
	svc := NewService(&fakeRepo{})

	result := svc.ValidateBulkOrder(context.Background(), "Неизвестное блюдо 2")
	if len(result.ValidItems) != 0 {
		t.Fatalf("valid items = %d, want 0", len(result.ValidItems))
	}
	assertContains(t, validationMessages(result.Errors), "не найдено в справочнике")
}

func TestServiceValidateBulkOrderReportsAmbiguousDishName(t *testing.T) {
	repo := &fakeRepo{
		resolveErrByName: map[string]error{"булочка": ErrDishCatalogItemAmbiguous},
	}
	svc := NewService(repo)

	result := svc.ValidateBulkOrder(context.Background(), "Булочка 2")
	if len(result.ValidItems) != 0 {
		t.Fatalf("valid items = %d, want 0", len(result.ValidItems))
	}
	assertContains(t, validationMessages(result.Errors), "найдено несколько раз")
}

func TestServiceListOrderTemplatesBuildsFromDishCatalog(t *testing.T) {
	repo := &fakeRepo{dishCatalog: []DishCatalogItem{
		{Code: "15542", Name: "Кокрок с картофелем", Theme: "КОКРОКИ"},
		{Code: "15544", Name: "Кокрок с творогом", Theme: "КОКРОКИ"},
		{Code: "15646", Name: "Самса с курицей", Theme: "САМСА И УЧПУЧМАК"},
	}}
	svc := NewService(repo)

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

func TestServiceListDishCatalogHidesCodes(t *testing.T) {
	repo := &fakeRepo{dishCatalog: []DishCatalogItem{
		{Code: "15542", Name: "Кокрок с картофелем", Theme: "КОКРОКИ", SortOrder: 2},
	}}
	svc := NewService(repo)

	items, err := svc.ListDishCatalog(context.Background())
	if err != nil {
		t.Fatalf("ListDishCatalog returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Кокрок с картофелем" || items[0].Theme != "КОКРОКИ" || items[0].SortOrder != 2 {
		t.Fatalf("catalog items = %#v", items)
	}
}

func TestServiceDeleteOrdersOlderThan(t *testing.T) {
	repo := &fakeRepo{deleteResult: 7}
	svc := NewService(repo)

	deleted, err := svc.DeleteOrdersOlderThan(context.Background(), time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC), 31*24*time.Hour)
	if err != nil {
		t.Fatalf("DeleteOrdersOlderThan returned error: %v", err)
	}
	if deleted != 7 {
		t.Fatalf("deleted = %d, want 7", deleted)
	}
	want := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	if !repo.deleteCutoff.Equal(want) {
		t.Fatalf("cutoff = %v, want %v", repo.deleteCutoff, want)
	}
}

func TestServiceCreateOrderRejectsPastFulfillmentDate(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	shopID := int64(10)
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)

	_, err := svc.CreateOrder(context.Background(), orderdomain.CreateOrderInput{
		Date:              now,
		FulfillmentDate:   now.AddDate(0, 0, -1),
		FromDepartmentID:  &shopID,
		CreatedByUsername: "shop",
		Items: []orderdomain.OrderItem{{
			Code:        "15635",
			ProductName: "Пирожок",
			Quantity:    1,
		}},
	})

	if !errors.Is(err, ErrFulfillmentDateInPast) {
		t.Fatalf("CreateOrder error = %v, want ErrFulfillmentDateInPast", err)
	}
	if repo.createCalled {
		t.Fatal("CreateOrder must not persist order with past fulfillment date")
	}
}

func TestServiceCreateOrderAllowsTodayFulfillmentDate(t *testing.T) {
	repo := &fakeRepo{
		departmentByID: map[int64]Department{
			10: {ID: 10, Code: "gagarina", Name: "Магазин Гагарина", Type: "shop"},
		},
	}
	svc := NewService(repo)
	shopID := int64(10)
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)

	_, err := svc.CreateOrder(context.Background(), orderdomain.CreateOrderInput{
		Date:              now,
		FulfillmentDate:   now,
		FromDepartmentID:  &shopID,
		CreatedByUsername: "shop",
		Items: []orderdomain.OrderItem{{
			Code:        "15635",
			ProductName: "Пирожок",
			Quantity:    1,
		}},
	})

	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}
	if !repo.createCalled {
		t.Fatal("CreateOrder should persist order with today's fulfillment date")
	}
}

func TestParseDefaultDishCatalogItems(t *testing.T) {
	items := parseDefaultDishCatalogItems(`
КОКРОКИ
15542 Кокрок с картофелем 0
15544 Кокрок с творогом 0

САМСА И УЧПУЧМАК
15646 Самса с курицей 0
broken line
`)
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3: %#v", len(items), items)
	}
	if items[0].Code != "15542" || items[0].Name != "Кокрок с картофелем" || items[0].Theme != "КОКРОКИ" || items[0].SortOrder != 1 {
		t.Fatalf("first item = %#v", items[0])
	}
	if items[2].Code != "15646" || items[2].Name != "Самса с курицей" || items[2].Theme != "САМСА И УЧПУЧМАК" || items[2].SortOrder != 3 {
		t.Fatalf("third item = %#v", items[2])
	}
}

func validationMessages(errs []orderdomain.BulkOrderValidationError) string {
	messages := make([]string, 0, len(errs))
	for _, item := range errs {
		messages = append(messages, item.Message)
	}
	return strings.Join(messages, "\n")
}

func assertContains(t *testing.T, value, needle string) {
	t.Helper()
	if !strings.Contains(value, needle) {
		t.Fatalf("%q does not contain %q", value, needle)
	}
}

// fakeRepo is an in-memory Repository test double.
type fakeRepo struct {
	dishExistsByCode map[string]bool
	dishErrorsByCode map[string]error
	resolveByName    map[string]DishCatalogItem
	resolveErrByName map[string]error
	dishCatalog      []DishCatalogItem
	deleteResult     int64
	deleteCutoff     time.Time
	departmentByID   map[int64]Department
	createCalled     bool
}

var _ Repository = (*fakeRepo)(nil)

func (f *fakeRepo) DishExistsByCode(_ context.Context, code string) (bool, error) {
	if err := f.dishErrorsByCode[code]; err != nil {
		return false, err
	}
	return f.dishExistsByCode[code], nil
}

func (f *fakeRepo) ResolveDishCatalogItem(_ context.Context, name string) (DishCatalogItem, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if err := f.resolveErrByName[key]; err != nil {
		return DishCatalogItem{}, err
	}
	if item, ok := f.resolveByName[key]; ok {
		return item, nil
	}
	return DishCatalogItem{}, ErrDishCatalogItemNotFound
}

func (f *fakeRepo) ListDishCatalog(_ context.Context) ([]DishCatalogItem, error) {
	out := make([]DishCatalogItem, len(f.dishCatalog))
	copy(out, f.dishCatalog)
	return out, nil
}

func (f *fakeRepo) DeleteOrdersOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	f.deleteCutoff = cutoff
	return f.deleteResult, nil
}

func (f *fakeRepo) CreateOrder(context.Context, CreateOrderRepositoryInput) (orderdomain.Order, error) {
	f.createCalled = true
	return orderdomain.Order{}, nil
}

func (f *fakeRepo) UpdateOrder(context.Context, UpdateOrderRepositoryInput) (orderdomain.Order, error) {
	return orderdomain.Order{}, nil
}

func (f *fakeRepo) GetOrderByNumber(context.Context, string) (orderdomain.Order, error) {
	return orderdomain.Order{}, nil
}

func (f *fakeRepo) ListOrders(context.Context, ListOrdersInput) (ListOrdersResult, error) {
	return ListOrdersResult{}, nil
}

func (f *fakeRepo) GetDepartmentByID(_ context.Context, id int64) (Department, error) {
	if department, ok := f.departmentByID[id]; ok {
		return department, nil
	}
	return Department{}, nil
}

func (f *fakeRepo) UpsertDishCatalogItem(context.Context, DishCatalogItem) error {
	return nil
}

func (f *fakeRepo) DeleteDishCatalogItem(context.Context, string) error {
	return nil
}
