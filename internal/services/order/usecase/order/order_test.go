package orderuc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
		categoryByID: map[int64]orderdomain.OrderCategory{
			1: {ID: 1, Code: "buns", Letter: "Б", Name: "Булочки", Color: "sky"},
		},
	}
	svc := NewService(repo)
	shopID := int64(10)
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)

	_, err := svc.CreateOrder(context.Background(), orderdomain.CreateOrderInput{
		Date:              now,
		FulfillmentDate:   now,
		FromDepartmentID:  &shopID,
		CategoryID:        1,
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

func TestServiceCreateProductionSheetNormalizesItems(t *testing.T) {
	repo := &fakeRepo{
		ordersByNumber: map[string]orderdomain.Order{
			"Г.Х.09.07.26.001": {
				Number: "Г.Х.09.07.26.001",
				Items: []orderdomain.OrderItem{
					{Code: "15702", ProductName: "Хлеб Бородино", Quantity: 8},
				},
			},
		},
	}
	svc := NewService(repo)

	sheet, err := svc.CreateProductionSheet(context.Background(), RecordProductionInput{
		ProducedByUsername: "baker",
		Orders: []OrderProductionInput{{
			Number: " Г.Х.09.07.26.001 ",
			Items: []ProducedItemInput{
				{ProductName: "хлеб бородино", ProducedQuantity: 7}, // имя матчится без регистра
			},
		}},
	})
	if err != nil {
		t.Fatalf("CreateProductionSheet returned error: %v", err)
	}
	if sheet.ID == 0 || len(repo.productionInputs) != 1 {
		t.Fatalf("sheet = %#v, inputs = %d", sheet, len(repo.productionInputs))
	}
	input := repo.productionInputs[0]
	if input.SheetID != 0 || input.ProducedByUsername != "baker" {
		t.Fatalf("input = %#v", input)
	}
	if len(input.Orders) != 1 || input.Orders[0].Number != "Г.Х.09.07.26.001" ||
		input.Orders[0].Items[0].ProductName != "Хлеб Бородино" || input.Orders[0].Items[0].ProducedQuantity != 7 {
		t.Fatalf("orders = %#v", input.Orders)
	}
}

func TestServiceCreateProductionSheetRejectsUnknownItemAndCancelled(t *testing.T) {
	repo := &fakeRepo{
		ordersByNumber: map[string]orderdomain.Order{
			"A": {Number: "A", Items: []orderdomain.OrderItem{{ProductName: "Хлеб", Quantity: 1}}},
			"C": {Number: "C", Cancelled: true, Items: []orderdomain.OrderItem{{ProductName: "Хлеб", Quantity: 1}}},
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateProductionSheet(context.Background(), RecordProductionInput{
		Orders: []OrderProductionInput{{Number: "A", Items: []ProducedItemInput{{ProductName: "Неизвестное", ProducedQuantity: 1}}}},
	})
	if err == nil {
		t.Fatal("CreateProductionSheet must reject unknown item")
	}

	_, err = svc.CreateProductionSheet(context.Background(), RecordProductionInput{
		Orders: []OrderProductionInput{{Number: "C", Items: []ProducedItemInput{{ProductName: "Хлеб", ProducedQuantity: 1}}}},
	})
	if err == nil {
		t.Fatal("CreateProductionSheet must reject cancelled order")
	}
	if len(repo.productionInputs) != 0 {
		t.Fatal("nothing must be persisted on validation errors")
	}
}

func TestServiceUpdateAndDeleteProductionSheetRequireExistingSheet(t *testing.T) {
	repo := &fakeRepo{
		sheetsByID: map[int64]struct{}{5: {}},
		ordersByNumber: map[string]orderdomain.Order{
			"A": {Number: "A", Items: []orderdomain.OrderItem{{ProductName: "Хлеб", Quantity: 1}}},
		},
	}
	svc := NewService(repo)

	input := RecordProductionInput{
		Orders: []OrderProductionInput{{Number: "A", Items: []ProducedItemInput{{ProductName: "Хлеб", ProducedQuantity: 2}}}},
	}
	if _, err := svc.UpdateProductionSheet(context.Background(), 99, input); !errors.Is(err, ErrProductionSheetNotFound) {
		t.Fatalf("UpdateProductionSheet(99) error = %v, want ErrProductionSheetNotFound", err)
	}
	if _, err := svc.UpdateProductionSheet(context.Background(), 5, input); err != nil {
		t.Fatalf("UpdateProductionSheet(5) error: %v", err)
	}
	if len(repo.productionInputs) != 1 || repo.productionInputs[0].SheetID != 5 {
		t.Fatalf("inputs = %#v", repo.productionInputs)
	}

	if err := svc.DeleteProductionSheet(context.Background(), 99, "baker"); !errors.Is(err, ErrProductionSheetNotFound) {
		t.Fatalf("DeleteProductionSheet(99) error = %v", err)
	}
	if err := svc.DeleteProductionSheet(context.Background(), 5, "baker"); err != nil {
		t.Fatalf("DeleteProductionSheet(5) error: %v", err)
	}
	if len(repo.productionDeleted) != 1 || repo.productionDeleted[0] != 5 {
		t.Fatalf("deleted = %#v", repo.productionDeleted)
	}
}

func TestServiceUpdateProductionSheetDeletesWhenAllValuesMatchOrder(t *testing.T) {
	repo := &fakeRepo{
		sheetsByID: map[int64]struct{}{5: {}},
		ordersByNumber: map[string]orderdomain.Order{
			"A": {Number: "A", Items: []orderdomain.OrderItem{{ProductName: "Хлеб", Quantity: 10}}},
		},
	}
	svc := NewService(repo)

	// Факт вернули к заявке — отклонений не осталось, документ удаляется.
	sheet, err := svc.UpdateProductionSheet(context.Background(), 5, RecordProductionInput{
		ProducedByUsername: "baker",
		Orders:             []OrderProductionInput{{Number: "A", Items: []ProducedItemInput{{ProductName: "Хлеб", ProducedQuantity: 10}}}},
	})
	if err != nil {
		t.Fatalf("UpdateProductionSheet returned error: %v", err)
	}
	if sheet.ID != 0 {
		t.Fatalf("sheet.ID = %d, want 0 (deleted)", sheet.ID)
	}
	if len(repo.productionDeleted) != 1 || repo.productionDeleted[0] != 5 {
		t.Fatalf("deleted = %#v, want [5]", repo.productionDeleted)
	}
	if len(repo.productionInputs) != 0 {
		t.Fatalf("save must not be called, inputs = %#v", repo.productionInputs)
	}
}

func TestServiceEnsureDefaultOrderTemplatesSeedsCategories(t *testing.T) {
	dir := t.TempDir()
	buns := filepath.Join(dir, "dishes.txt")
	bread := filepath.Join(dir, "bread.txt")
	if err := os.WriteFile(buns, []byte("БУЛОЧКИ\n15664 Булочка Улитка 0\n15667 Плюшка московская 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bread, []byte("ХЛЕБ\n15702 Хлеб Бородино 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := &fakeRepo{
		categoryByID: map[int64]orderdomain.OrderCategory{
			1: {ID: 1, Code: "bread", Letter: "Х", Name: "Хлеб"},
			2: {ID: 2, Code: "buns", Letter: "Б", Name: "Булочки"},
		},
	}
	svc := NewService(repo)

	result, err := svc.EnsureDefaultOrderTemplates(context.Background(),
		CatalogSeed{Path: buns, CategoryCode: "buns"},
		CatalogSeed{Path: bread, CategoryCode: "bread"},
		CatalogSeed{Path: filepath.Join(dir, "missing.txt"), CategoryCode: "buns"}, // не ошибка
	)
	if err != nil {
		t.Fatalf("EnsureDefaultOrderTemplates returned error: %v", err)
	}
	if result.CatalogItems != 3 || len(repo.upserted) != 3 {
		t.Fatalf("catalog items = %d, upserted = %d, want 3/3", result.CatalogItems, len(repo.upserted))
	}

	byCode := map[string]DishCatalogItem{}
	for _, item := range repo.upserted {
		byCode[item.Code] = item
	}
	if item := byCode["15664"]; item.CategoryID == nil || *item.CategoryID != 2 || item.SortOrder != 1 {
		t.Fatalf("булочка = %#v, want category 2 sort 1", item)
	}
	// Сквозная нумерация: хлебный файл продолжает счёт после булочек.
	if item := byCode["15702"]; item.CategoryID == nil || *item.CategoryID != 1 || item.SortOrder != 3 {
		t.Fatalf("хлеб = %#v, want category 1 sort 3", item)
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
	dishExistsByCode  map[string]bool
	dishErrorsByCode  map[string]error
	resolveByName     map[string]DishCatalogItem
	resolveErrByName  map[string]error
	dishCatalog       []DishCatalogItem
	deleteResult      int64
	deleteCutoff      time.Time
	departmentByID    map[int64]Department
	categoryByID      map[int64]orderdomain.OrderCategory
	createCalled      bool
	ordersByNumber    map[string]orderdomain.Order
	sheetsByID        map[int64]struct{}
	productionInputs  []SaveProductionSheetInput
	productionDeleted []int64
	upserted          []DishCatalogItem
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

func (f *fakeRepo) NextOrderNumber(_ context.Context, input NextOrderNumberInput) (string, error) {
	return "TEST." + input.Category.Letter + ".001", nil
}

func (f *fakeRepo) GetOrderByNumber(_ context.Context, number string) (orderdomain.Order, error) {
	if order, ok := f.ordersByNumber[number]; ok {
		return order, nil
	}
	if f.ordersByNumber != nil {
		return orderdomain.Order{}, ErrProductionOrderNotFound
	}
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

func (f *fakeRepo) SaveProductionSheet(_ context.Context, input SaveProductionSheetInput) (orderdomain.ProductionSheet, error) {
	f.productionInputs = append(f.productionInputs, input)
	return orderdomain.ProductionSheet{ID: max(input.SheetID, 1)}, nil
}

func (f *fakeRepo) ListProductionSheets(context.Context) ([]orderdomain.ProductionSheet, error) {
	return nil, nil
}

func (f *fakeRepo) GetProductionSheet(_ context.Context, id int64) (orderdomain.ProductionSheet, error) {
	if _, ok := f.sheetsByID[id]; ok {
		return orderdomain.ProductionSheet{ID: id}, nil
	}
	return orderdomain.ProductionSheet{}, ErrProductionSheetNotFound
}

func (f *fakeRepo) DeleteProductionSheet(_ context.Context, id int64, _ string) error {
	f.productionDeleted = append(f.productionDeleted, id)
	return nil
}

func (f *fakeRepo) ListOrderCategories(context.Context) ([]orderdomain.OrderCategory, error) {
	out := make([]orderdomain.OrderCategory, 0, len(f.categoryByID))
	for _, category := range f.categoryByID {
		out = append(out, category)
	}
	return out, nil
}

func (f *fakeRepo) GetOrderCategoryByID(_ context.Context, id int64) (orderdomain.OrderCategory, error) {
	if category, ok := f.categoryByID[id]; ok {
		return category, nil
	}
	return orderdomain.OrderCategory{}, ErrCategoryNotFound
}

func (f *fakeRepo) CreateOrderCategory(_ context.Context, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error) {
	return input, nil
}

func (f *fakeRepo) UpdateOrderCategory(_ context.Context, _ int64, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error) {
	return input, nil
}

func (f *fakeRepo) DeleteOrderCategory(context.Context, int64) error {
	return nil
}

func (f *fakeRepo) CountDishesByCategoryID(context.Context, int64) (int64, error) {
	return 0, nil
}

func (f *fakeRepo) UpsertDishCatalogItem(_ context.Context, item DishCatalogItem) error {
	f.upserted = append(f.upserted, item)
	return nil
}

func (f *fakeRepo) UpdateDishCatalogItem(_ context.Context, _ string, item DishCatalogItem) (DishCatalogItem, error) {
	return item, nil
}

func (f *fakeRepo) SearchAvailableDishes(context.Context, string, int) ([]orderdomain.AvailableDish, error) {
	return nil, nil
}

func (f *fakeRepo) SetDishCatalogSortOrder(context.Context, string, int64) error {
	return nil
}

func (f *fakeRepo) DeleteDishCatalogItem(context.Context, string) error {
	return nil
}

func (f *fakeRepo) SetOrderFavorite(context.Context, string, bool) (orderdomain.Order, error) {
	return orderdomain.Order{}, nil
}

func (f *fakeRepo) CancelOrder(context.Context, string, string) (orderdomain.Order, error) {
	return orderdomain.Order{}, nil
}

func (f *fakeRepo) RestoreOrder(context.Context, string) (orderdomain.Order, error) {
	return orderdomain.Order{}, nil
}
