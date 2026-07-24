package web

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"bakery/frontend/internal/backend"
	"bakery/internal/inbound/api/contract"
)

func TestFragmentRequestsOverrideShellSelection(t *testing.T) {
	t.Parallel()
	fragmentRequest := regexp.MustCompile(`<[^>]+hx-get="/fragments/[^>]+>`)
	found := 0

	err := fs.WalkDir(frontendFiles, "templates", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		source, err := frontendFiles.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range fragmentRequest.FindAllString(string(source), -1) {
			found++
			if !strings.Contains(match, `hx-select="unset"`) {
				t.Errorf("%s: fragment request inherits shell hx-select: %s", path, match)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk templates: %v", err)
	}
	if found == 0 {
		t.Fatal("no fragment requests found in templates")
	}
}

func TestTemplatesRenderEveryView(t *testing.T) {
	t.Parallel()
	client := backend.New("http://127.0.0.1:1", time.Second)
	srv, err := newServer(client, client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	categoryID := int64(1)
	sheetID := int64(2)
	category := contract.Category{ID: categoryID, Name: "Хлеб", Letter: "Х", Color: "amber"}
	dish := contract.Dish{Code: "dish", Name: "Багет", Theme: "Хлеб", CategoryID: &categoryID}
	order := contract.Order{
		Number: "Г.Х.22.07.26.001", Category: &category, FulfillmentDate: "2026-07-23",
		CreatedAt: time.Now().Format(time.RFC3339), CreatedByUsername: "tester", ProductionSheetID: &sheetID,
		Items: []contract.OrderItem{{Code: "dish", ProductName: "Багет", Quantity: 3, ProductionQuantity: 3}},
	}
	sheet := contract.ProductionSheet{ID: sheetID, CreatedAt: time.Now().Format(time.RFC3339), CreatedByUsername: "tester", OrderNumbers: []string{order.Number}}
	viewer := &contract.Me{Role: "admin", TelegramUsername: "tester"}
	shopViewer := &contract.Me{Role: "shop", TelegramUsername: "tester"}

	tests := []page{
		{View: "login", Data: loginData{Next: "/orders"}},
		{View: "error", Viewer: viewer, Error: "Ошибка"},
		{View: "me", Viewer: viewer},
		{View: "orders", Viewer: shopViewer, Data: ordersData{Page: contract.OrdersPage{Items: []contract.Order{order}, Page: 2, TotalPages: 3}, Categories: []contract.Category{category}}},
		{View: "order-detail", Viewer: viewer, Data: orderDetailData{Order: order, Groups: []orderItemGroup{{Name: "Хлеб", Items: order.Items}}}},
		{View: "order-form", Viewer: viewer, Data: orderFormData{Mode: "create", Categories: []contract.Category{category}, Groups: []editorGroup{{Name: "Хлеб", Items: []editorItem{{Dish: dish}}}}}},
		{View: "selection", Viewer: viewer, Data: selectionData{Orders: []contract.Order{order}, Rows: buildProductionRows([]contract.Order{order}, nil)}},
		{View: "orders-table", Viewer: viewer, Data: buildOrdersTable([]contract.Order{order}, []contract.Dish{dish}, []contract.Category{category}, categoryID, time.Now(), time.Now())},
		{View: "production", Viewer: viewer, Data: productionJournalData{Categories: []contract.Category{category}, Rows: []productionJournalRow{{Date: "2026-07-22", Cells: [][]productionSheetView{{{Sheet: sheet, Category: &category}}}}}, Count: 1}},
		{View: "production-detail", Viewer: viewer, Data: productionEditorData{Sheet: &sheet, Orders: []contract.Order{order}, Rows: buildProductionRows([]contract.Order{order}, nil)}},
		{View: "calculator", Viewer: viewer, Data: calculatorData{Categories: []contract.Category{category}, Groups: []calculatorGroup{{Name: "Хлеб", Items: []calculatorItem{{Dish: dish}}}}}},
		{
			View:   "admin-users",
			Viewer: viewer,
			Data: adminUsersData{Users: []adminUserView{
				{User: contract.User{ID: 1, Username: "tester", Role: "admin"}},
			}},
		},
		{View: "admin-dishes", Viewer: viewer, Data: adminDishesData{Dishes: []contract.Dish{dish}, Categories: []contract.Category{category}}},
	}

	for _, value := range tests {
		value.Title = value.View
		value.CurrentPath = "/" + value.View
		value.CSRF = "csrf"
		value.Today = "2026-07-22"
		if err := srv.templates.ExecuteTemplate(io.Discard, "layout", value); err != nil {
			t.Errorf("render %s: %v", value.View, err)
		}
	}
}

func TestSessionCookieRoundTrip(t *testing.T) {
	t.Parallel()
	srv := &server{}
	request := httptest.NewRequest("GET", "http://example.test/orders", nil)
	response := httptest.NewRecorder()
	srv.setSession(response, request, "Bearer signed-token", time.Now().Add(time.Hour))

	result := response.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	request.AddCookie(cookies[0])
	if got := credentials(request); got != "Bearer signed-token" {
		t.Fatalf("credentials = %q", got)
	}
}

func TestSafeNextRejectsExternalURL(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"https://example.com", "//example.com", "javascript:alert(1)"} {
		if got := safeNext(value); got != "/orders" {
			t.Errorf("safeNext(%q) = %q", value, got)
		}
	}
	if got := safeNext("/orders/table?start=2026-07-22"); got != "/orders/table?start=2026-07-22" {
		t.Errorf("safe local next = %q", got)
	}
}
