package orderrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"bakery/internal/outbound/db/dbtest"
	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/apperr"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
)

type fixture struct {
	db       *sql.DB
	repo     *OrderRepository
	shop     orderuc.Department
	category orderdomain.OrderCategory
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db := dbtest.Open(t)
	ctx := context.Background()
	queries := sqlc.New(db)
	shop, err := queries.GetDepartmentByCode(ctx, "gagarina")
	if err != nil {
		t.Fatalf("load shop: %v", err)
	}
	repo := New(queries, db)
	categories, err := repo.ListOrderCategories(ctx)
	if err != nil {
		t.Fatalf("load categories: %v", err)
	}
	var bread orderdomain.OrderCategory
	for _, category := range categories {
		if category.Code == "bread" {
			bread = category
		}
	}
	if bread.ID == 0 {
		t.Fatal("bread category is not seeded")
	}
	return fixture{
		db:       db,
		repo:     repo,
		shop:     orderuc.Department{ID: shop.ID, Code: shop.Code, Name: shop.Name, Type: shop.Type},
		category: bread,
	}
}

func (f fixture) createOrder(t *testing.T, fulfillment time.Time, dedupKey string, items ...orderdomain.OrderItem) orderdomain.Order {
	t.Helper()
	order, err := f.repo.CreateOrder(context.Background(), f.createInput(fulfillment, dedupKey, items...))
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	return order
}

func (f fixture) createInput(fulfillment time.Time, dedupKey string, items ...orderdomain.OrderItem) orderuc.CreateOrderRepositoryInput {
	if len(items) == 0 {
		items = []orderdomain.OrderItem{{ProductName: "Хлеб Бородинский", Quantity: 10}}
	}
	createdAt := time.Date(2026, 11, 21, 9, 30, 0, 0, time.UTC)
	shopID := f.shop.ID
	return orderuc.CreateOrderRepositoryInput{
		Input: orderdomain.CreateOrderInput{
			Items:             items,
			Location:          f.shop.Name,
			FromDepartmentID:  &shopID,
			CategoryID:        f.category.ID,
			CreatedByUsername: "shop-user",
		},
		Source:          f.shop,
		Category:        f.category,
		CreatedAt:       createdAt,
		FulfillmentDate: fulfillment,
		CounterDay:      orderdomain.NewOrderService().OrderCounterDay(createdAt),
		DedupKey:        dedupKey,
	}
}

func (f fixture) outboxTypes(t *testing.T) []string {
	t.Helper()
	rows, err := f.db.Query(`SELECT event_type, payload FROM order_outbox ORDER BY id`)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var types []string
	for rows.Next() {
		var eventType, payload string
		if err := rows.Scan(&eventType, &payload); err != nil {
			t.Fatalf("scan outbox row: %v", err)
		}
		var event struct {
			Order orderdomain.Order `json:"order"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			t.Fatalf("decode outbox payload: %v", err)
		}
		if event.Order.Number == "" {
			t.Fatalf("outbox payload has no order: %s", payload)
		}
		types = append(types, eventType)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return types
}

func (f fixture) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func day(offset int) time.Time {
	return time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC).AddDate(0, 0, offset)
}

func TestCreateOrderNumbersSequentiallyAndWritesCreatedEventToOutbox(t *testing.T) {
	f := newFixture(t)

	first := f.createOrder(t, day(0), "")
	second := f.createOrder(t, day(0), "")

	wantFirst := orderdomain.NewOrderService().BuildOrderNumber(f.shop.Code, f.shop.Name, f.category.Letter, time.Date(2026, 11, 21, 9, 30, 0, 0, time.UTC), 1)
	if first.Number != wantFirst {
		t.Fatalf("first number = %q, want %q", first.Number, wantFirst)
	}
	if first.Number == second.Number {
		t.Fatalf("numbers collide: %q", first.Number)
	}
	if got := f.outboxTypes(t); len(got) != 2 || got[0] != orderdomain.EventOrderCreated || got[1] != orderdomain.EventOrderCreated {
		t.Fatalf("outbox types = %v", got)
	}

	loaded, err := f.repo.GetOrderByNumber(context.Background(), first.Number)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.FulfillmentDate.Equal(day(0)) {
		t.Fatalf("fulfillment date = %v, want %v", loaded.FulfillmentDate, day(0))
	}
	if loaded.Category == nil || loaded.Category.Code != "bread" {
		t.Fatalf("category = %+v", loaded.Category)
	}
}

func TestCreateOrderDedupKeyReturnsExistingOrder(t *testing.T) {
	f := newFixture(t)

	first := f.createOrder(t, day(0), "same-submit")
	again := f.createOrder(t, day(0), "same-submit")

	if again.Number != first.Number {
		t.Fatalf("dedup returned %q, want %q", again.Number, first.Number)
	}
	if n := f.count(t, "orders"); n != 1 {
		t.Fatalf("orders = %d, want 1", n)
	}
	if got := f.outboxTypes(t); len(got) != 1 {
		t.Fatalf("outbox = %v, want exactly one created event", got)
	}
}

func TestCreateOrderRollsBackWhenOutboxWriteFails(t *testing.T) {
	f := newFixture(t)
	if _, err := f.db.Exec(`DROP TABLE order_outbox`); err != nil {
		t.Fatal(err)
	}

	_, err := f.repo.CreateOrder(context.Background(), f.createInput(day(0), ""))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, table := range []string{"orders", "order_items", "order_counters"} {
		if n := f.count(t, table); n != 0 {
			t.Fatalf("%s = %d rows after rollback", table, n)
		}
	}
}

func TestListOrdersFiltersByFulfillmentDate(t *testing.T) {
	f := newFixture(t)
	for offset := range 3 {
		f.createOrder(t, day(offset), "")
	}
	ctx := context.Background()

	exact, err := f.repo.ListOrders(ctx, orderuc.ListOrdersInput{Limit: 50, FulfillmentDate: day(1)})
	if err != nil {
		t.Fatal(err)
	}
	if exact.Total != 1 || len(exact.Orders) != 1 || !exact.Orders[0].FulfillmentDate.Equal(day(1)) {
		t.Fatalf("exact filter = total %d, orders %d", exact.Total, len(exact.Orders))
	}

	ranged, err := f.repo.ListOrders(ctx, orderuc.ListOrdersInput{Limit: 50, FulfillmentFrom: day(1), FulfillmentTo: day(2)})
	if err != nil {
		t.Fatal(err)
	}
	if ranged.Total != 2 || len(ranged.Orders) != 2 {
		t.Fatalf("range filter = total %d, orders %d", ranged.Total, len(ranged.Orders))
	}

	otherCategory := f.category.ID + 1000
	none, err := f.repo.ListOrders(ctx, orderuc.ListOrdersInput{Limit: 50, CategoryID: &otherCategory})
	if err != nil {
		t.Fatal(err)
	}
	if none.Total != 0 {
		t.Fatalf("category filter total = %d", none.Total)
	}
}

func TestProductionSheetMatchesCyrillicNamesCaseInsensitively(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	order := f.createOrder(t, day(0), "", orderdomain.OrderItem{ProductName: "Хлеб Бородинский", Quantity: 10})
	loaded := 10.0

	sheet, err := f.repo.SaveProductionSheet(ctx, orderuc.SaveProductionSheetInput{
		ProducedByUsername: "baker",
		Orders: []orderuc.OrderProductionInput{{
			Number: order.Number,
			Items: []orderuc.ProducedItemInput{{
				ProductName:      " ХЛЕБ БОРОДИНСКИЙ ",
				LoadedQuantity:   &loaded,
				ProducedQuantity: 8,
				IsDeviation:      true,
				Reason:           "подгорело",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("save production sheet: %v", err)
	}
	if len(sheet.Items) != 1 || sheet.Items[0].ProducedQuantity != 8 {
		t.Fatalf("sheet items = %+v", sheet.Items)
	}

	decorated, err := f.repo.GetOrderByNumber(ctx, order.Number)
	if err != nil {
		t.Fatal(err)
	}
	item := decorated.Items[0]
	if item.ProducedQuantity == nil || *item.ProducedQuantity != 8 || item.ProducedReason != "подгорело" {
		t.Fatalf("decorated item = %+v", item)
	}
	if decorated.ProductionSheetID == nil || *decorated.ProductionSheetID != sheet.ID {
		t.Fatalf("production sheet id = %v, want %d", decorated.ProductionSheetID, sheet.ID)
	}

	sheets, err := f.repo.ListProductionSheets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheets) != 1 || len(sheets[0].OrderNumbers) != 1 || sheets[0].OrderNumbers[0] != order.Number || sheets[0].ItemCount != 1 {
		t.Fatalf("sheets = %+v", sheets)
	}

	if got := f.outboxTypes(t); len(got) != 2 || got[1] != orderdomain.EventOrderProduced {
		t.Fatalf("outbox types = %v", got)
	}
}

func TestListProductionSheetsCategoryFromFirstOrderByNumber(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	queries := sqlc.New(f.db)
	categories, err := f.repo.ListOrderCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var otherCategoryID int64
	for _, category := range categories {
		if category.ID != f.category.ID {
			otherCategoryID = category.ID
			break
		}
	}
	if otherCategoryID == 0 {
		t.Fatal("second category is not seeded")
	}
	tests := []struct {
		name       string
		withOrders bool
		categoryID *int64
	}{
		{name: "mixed categories", withOrders: true, categoryID: &otherCategoryID},
		{name: "uncategorized first order", withOrders: true},
		{name: "empty batch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheet, err := queries.CreateProductionSheet(ctx, "baker")
			if err != nil {
				t.Fatal(err)
			}
			var wantNumbers []string
			if tt.withOrders {
				for _, prefix := range []string{"Z", "A"} {
					order := f.createOrder(t, day(0), "")
					row, err := queries.GetOrderByNumber(ctx, order.Number)
					if err != nil {
						t.Fatal(err)
					}
					categoryID := &f.category.ID
					if prefix == "A" {
						categoryID = tt.categoryID
					}
					number := fmt.Sprintf("%s-%d", prefix, sheet.ID)
					if _, err := f.db.ExecContext(ctx, `UPDATE orders SET number = ?, category_id = ? WHERE id = ?`, number, categoryID, row.ID); err != nil {
						t.Fatal(err)
					}
					if err := queries.InsertProductionSheetOrder(ctx, sqlc.InsertProductionSheetOrderParams{SheetID: sheet.ID, OrderID: row.ID}); err != nil {
						t.Fatal(err)
					}
				}
				wantNumbers = []string{fmt.Sprintf("A-%d", sheet.ID), fmt.Sprintf("Z-%d", sheet.ID)}
			}
			sheets, err := f.repo.ListProductionSheets(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(sheets) == 0 || sheets[0].ID != sheet.ID {
				t.Fatalf("newest sheet missing: %+v", sheets)
			}
			got := sheets[0]
			if !slices.Equal(got.OrderNumbers, wantNumbers) {
				t.Fatalf("order numbers = %v, want %v", got.OrderNumbers, wantNumbers)
			}
			if tt.categoryID == nil {
				if got.CategoryID != nil {
					t.Fatalf("category ID = %d, want nil", *got.CategoryID)
				}
			} else if got.CategoryID == nil || *got.CategoryID != *tt.categoryID {
				t.Fatalf("sheet = %+v, want category ID %d", got, *tt.categoryID)
			}
		})
	}
}

func TestProductionSheetRejectsOrderAlreadyInAnotherSheet(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	order := f.createOrder(t, day(0), "")
	loaded := 10.0
	input := orderuc.SaveProductionSheetInput{
		ProducedByUsername: "baker",
		Orders: []orderuc.OrderProductionInput{{
			Number: order.Number,
			Items:  []orderuc.ProducedItemInput{{ProductName: "Хлеб Бородинский", LoadedQuantity: &loaded, ProducedQuantity: 10}},
		}},
	}
	if _, err := f.repo.SaveProductionSheet(ctx, input); err != nil {
		t.Fatalf("first sheet: %v", err)
	}

	_, err := f.repo.SaveProductionSheet(ctx, input)
	if appErr := apperr.As(err); appErr == nil || appErr.Code != "order.production_exists" {
		t.Fatalf("second sheet error = %v, want order.production_exists", err)
	}
}

func TestProductionSheetOrderInsertErrorMapsUniqueOrderViolation(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	order := f.createOrder(t, day(0), "")
	orderID := mustOrderID(t, f, order.Number)
	queries := sqlc.New(f.db)
	first, err := queries.CreateProductionSheet(ctx, "baker")
	if err != nil {
		t.Fatal(err)
	}
	second, err := queries.CreateProductionSheet(ctx, "baker")
	if err != nil {
		t.Fatal(err)
	}
	if err := queries.InsertProductionSheetOrder(ctx, sqlc.InsertProductionSheetOrderParams{SheetID: first.ID, OrderID: orderID}); err != nil {
		t.Fatal(err)
	}

	insertErr := queries.InsertProductionSheetOrder(ctx, sqlc.InsertProductionSheetOrderParams{SheetID: second.ID, OrderID: orderID})
	got := productionSheetOrderInsertError(fmt.Errorf("insert: %w", insertErr), order.Number)
	if appErr := apperr.As(got); appErr == nil || appErr.Kind != apperr.KindConflict || appErr.Code != "order.production_exists" {
		t.Fatalf("mapped error = %v", got)
	}

	other := errors.New("disk full")
	if got := productionSheetOrderInsertError(other, order.Number); !errors.Is(got, other) {
		t.Fatalf("other error = %v, want wrapped original", got)
	}
}

func TestCancelAndRestoreWriteOutboxEvents(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	order := f.createOrder(t, day(0), "dedup")

	cancelled, err := f.repo.CancelOrder(ctx, order.Number, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if !cancelled.Cancelled || cancelled.CancelledByUsername != "admin" {
		t.Fatalf("cancelled = %+v", cancelled)
	}
	restored, err := f.repo.RestoreOrder(ctx, order.Number)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Cancelled {
		t.Fatal("order still cancelled after restore")
	}

	want := []string{orderdomain.EventOrderCreated, orderdomain.EventOrderCancelled, orderdomain.EventOrderRestored}
	got := f.outboxTypes(t)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("outbox types = %v, want %v", got, want)
	}

	if _, err := f.repo.CancelOrder(ctx, "missing", "admin"); apperr.As(err) == nil {
		t.Fatalf("cancel missing order error = %v, want not found", err)
	}
}

func TestDeleteOrdersOlderThanKeepsFavorites(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	plain := f.createOrder(t, day(0), "")
	favorite := f.createOrder(t, day(0), "")
	if _, err := f.repo.SetOrderFavorite(ctx, favorite.Number, true); err != nil {
		t.Fatal(err)
	}

	deleted, err := f.repo.DeleteOrdersOlderThan(ctx, time.Date(2026, 11, 21, 9, 30, 0, 1, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := f.repo.GetOrderByNumber(ctx, plain.Number); err == nil {
		t.Fatal("plain order survived cleanup")
	}
	if _, err := f.repo.GetOrderByNumber(ctx, favorite.Number); err != nil {
		t.Fatalf("favorite order removed: %v", err)
	}
}

func TestSearchAvailableDishesFoldsCyrillicCase(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if _, err := f.db.Exec(`INSERT INTO iiko_products (id, code, name, type, raw_json) VALUES ('p1', '101', 'Булочка с Маком', 'DISH', '{}')`); err != nil {
		t.Fatal(err)
	}

	found, err := f.repo.SearchAvailableDishes(ctx, "МАК", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Code != "101" {
		t.Fatalf("found = %+v", found)
	}

	empty, err := f.repo.SearchAvailableDishes(ctx, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty query returned %+v", empty)
	}
}

func TestOrderCategoryMonitorCodesRoundTrip(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if fmt.Sprint(f.category.MonitorCodes) != "[21429 17642 21212 21192]" {
		t.Fatalf("seeded bread monitor codes = %v", f.category.MonitorCodes)
	}

	created, err := f.repo.CreateOrderCategory(ctx, orderdomain.OrderCategory{Code: "cakes", Letter: "Т", Name: "Торты", Color: "rose"})
	if err != nil {
		t.Fatal(err)
	}
	if created.MonitorCodes == nil || len(created.MonitorCodes) != 0 {
		t.Fatalf("empty monitor codes = %#v", created.MonitorCodes)
	}
	updated, err := f.repo.UpdateOrderCategory(ctx, created.ID, orderdomain.OrderCategory{Letter: "Т", Name: "Торты", Color: "rose", MonitorCodes: []string{"1", "2"}})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(updated.MonitorCodes) != "[1 2]" {
		t.Fatalf("updated monitor codes = %v", updated.MonitorCodes)
	}
}

func TestOrderDraftRoundTripAndCleanup(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	saved, err := f.repo.SaveOrderDraft(ctx, orderuc.SaveOrderDraftRepositoryInput{
		CreatedByUsername: "shop-user",
		CategoryID:        f.category.ID,
		FromDepartmentID:  f.shop.ID,
		Items:             []orderdomain.OrderItem{{ProductName: "Батон", Quantity: 3}},
		FulfillmentDate:   day(1),
		Comments:          orderdomain.OrderComments{General: "к утру"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.FulfillmentDate.Equal(day(1)) || len(saved.Items) != 1 || saved.Comments.General != "к утру" {
		t.Fatalf("saved draft = %+v", saved)
	}

	deleted, err := f.repo.DeleteOrderDraftsOlderThan(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted drafts = %d, want 1", deleted)
	}
}

func mustOrderID(t *testing.T, f fixture, number string) int64 {
	t.Helper()
	row, err := sqlc.New(f.db).GetOrderByNumber(context.Background(), number)
	if err != nil {
		t.Fatal(err)
	}
	return row.ID
}

func TestDeletedIDsAreNeverReused(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	queries := sqlc.New(f.db)

	sheet, err := queries.CreateProductionSheet(ctx, "baker")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.DeleteProductionSheet(ctx, sheet.ID, "baker"); err != nil {
		t.Fatal(err)
	}
	next, err := queries.CreateProductionSheet(ctx, "baker")
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == sheet.ID {
		t.Fatalf("production sheet id %d reused after delete", sheet.ID)
	}

	order := f.createOrder(t, day(0), "")
	orderID := mustOrderID(t, f, order.Number)
	if _, err := f.db.Exec(`DELETE FROM orders WHERE id = ?`, orderID); err != nil {
		t.Fatal(err)
	}
	again := f.createOrder(t, day(0), "")
	if mustOrderID(t, f, again.Number) == orderID {
		t.Fatalf("order id %d reused after delete", orderID)
	}
}

func TestDeletePublishedOrderEventsKeepsRecentAndUndelivered(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	old := now.Add(-40 * 24 * time.Hour)
	recent := now.Add(-24 * time.Hour)
	rows := []struct {
		aggregate string
		created   time.Time
		published *time.Time
	}{
		{"delivered-old", old, &old},
		{"delivered-recent", recent, &recent},
		{"undelivered-old", old, nil},
	}
	for _, row := range rows {
		if _, err := f.db.ExecContext(ctx,
			`INSERT INTO order_outbox (aggregate_id, event_type, payload, created_at, published_at) VALUES (?, 'order.created', '{}', ?, ?)`,
			row.aggregate, row.created, row.published,
		); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := f.repo.DeletePublishedOrderEventsBefore(ctx, now.Add(-31*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	remaining, err := f.db.QueryContext(ctx, `SELECT aggregate_id FROM order_outbox ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = remaining.Close() }()
	var got []string
	for remaining.Next() {
		var aggregate string
		if err := remaining.Scan(&aggregate); err != nil {
			t.Fatal(err)
		}
		got = append(got, aggregate)
	}
	if err := remaining.Err(); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != fmt.Sprint([]string{"delivered-recent", "undelivered-old"}) {
		t.Fatalf("remaining = %v", got)
	}
}
