package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
	monitoringdomain "bakery/internal/services/monitor/domain"
)

// fakeBackend implements application.Queries and application.Commands in memory,
// so the E2E tests exercise the real router, middlewares, session and CSRF
// handling end to end without a live worker API. Only the methods the flows
// touch return meaningful data; the rest return zero values.
type fakeBackend struct {
	// viewers maps an Authorization credential to the resolved viewer.
	// After login the credential is "Bearer <token>".
	viewers map[application.Credentials]contract.Me
	// loginToken is handed out by Login for the recognised username.
	loginToken      string
	loginErr        error
	createdOrder    contract.Order
	updatedOrder    contract.Order
	orders          map[string]contract.Order
	sheets          []contract.ProductionSheet
	sheet           contract.ProductionSheet
	batchMonitor    contract.BatchMonitor
	categories      []contract.Category
	orderFilters    application.OrderFilters
	orderPages      map[int]contract.OrdersPage
	orderCalls      []application.OrderFilters
	drafts          []contract.OrderDraft
	draftByCategory map[int64]contract.OrderDraft

	// call captures for the write-flow assertions below.
	createOrderCalls     []contract.OrderWrite
	updateOrderCalls     []orderUpdateCall
	cancelOrderCalls     []string
	restoreOrderCalls    []string
	favoriteCalls        []favoriteCall
	savedDraftWrites     []contract.OrderWrite
	deletedDraftCategory []int64
}

type orderUpdateCall struct {
	Number string
	Write  contract.OrderWrite
}

type favoriteCall struct {
	Number   string
	Favorite bool
}

func (f *fakeBackend) Health(context.Context) error { return nil }

func (f *fakeBackend) Me(_ context.Context, cred application.Credentials) (contract.Me, error) {
	if me, ok := f.viewers[cred]; ok {
		return me, nil
	}
	return contract.Me{}, &application.Error{Status: http.StatusUnauthorized, Message: "нет доступа"}
}

func (f *fakeBackend) Departments(context.Context, application.Credentials, string) ([]contract.Department, error) {
	return []contract.Department{{ID: 1, Code: "shop-1", Name: "Магазин 1", Type: "shop"}}, nil
}
func (f *fakeBackend) Catalog(context.Context, application.Credentials) ([]contract.Dish, error) {
	return []contract.Dish{{Code: "dish", Name: "Багет", Theme: "Хлеб"}}, nil
}
func (f *fakeBackend) Categories(context.Context, application.Credentials) ([]contract.Category, error) {
	if f.categories != nil {
		return f.categories, nil
	}
	return []contract.Category{{ID: 1, Name: "Хлеб", Letter: "Х", Color: "amber"}}, nil
}
func (f *fakeBackend) Orders(_ context.Context, _ application.Credentials, filters application.OrderFilters) (contract.OrdersPage, error) {
	f.orderFilters = filters
	f.orderCalls = append(f.orderCalls, filters)
	if page, ok := f.orderPages[filters.Page]; ok {
		return page, nil
	}
	return contract.OrdersPage{Items: nil, Total: 0}, nil
}
func (f *fakeBackend) Order(_ context.Context, _ application.Credentials, number string) (contract.Order, error) {
	return f.orders[number], nil
}
func (f *fakeBackend) ProductionSheets(context.Context, application.Credentials) ([]contract.ProductionSheet, error) {
	return f.sheets, nil
}
func (f *fakeBackend) ProductionSheet(context.Context, application.Credentials, int64) (contract.ProductionSheet, error) {
	return f.sheet, nil
}
func (f *fakeBackend) OrderMonitor(context.Context, application.Credentials, string) (contract.OrderMonitor, error) {
	return contract.OrderMonitor{}, nil
}
func (f *fakeBackend) BatchMonitor(context.Context, application.Credentials, []string) (contract.BatchMonitor, error) {
	return f.batchMonitor, nil
}
func (f *fakeBackend) Users(context.Context, application.Credentials) ([]contract.User, error) {
	return []contract.User{{ID: 1, Username: "admin", Role: "admin"}}, nil
}
func (f *fakeBackend) AdminDepartments(context.Context, application.Credentials) ([]contract.Department, error) {
	return nil, nil
}
func (f *fakeBackend) Dishes(context.Context, application.Credentials) ([]contract.Dish, error) {
	return nil, nil
}
func (f *fakeBackend) AvailableDishes(context.Context, application.Credentials, string) ([]contract.AvailableDish, error) {
	return nil, nil
}
func (f *fakeBackend) DishTechCards(context.Context, application.Credentials) ([]contract.TechCardCategory, error) {
	return nil, nil
}

func (f *fakeBackend) Login(_ context.Context, username, _, _ string) (contract.LoginResponse, error) {
	if f.loginErr != nil {
		return contract.LoginResponse{}, f.loginErr
	}
	return contract.LoginResponse{Token: f.loginToken, ExpiresAt: 1 << 40}, nil
}
func (f *fakeBackend) CreateOrder(_ context.Context, _ application.Credentials, body contract.OrderWrite) (contract.Order, error) {
	f.createOrderCalls = append(f.createOrderCalls, body)
	order := f.createdOrder
	if f.orders == nil {
		f.orders = make(map[string]contract.Order)
	}
	f.orders[order.Number] = order
	return order, nil
}
func (f *fakeBackend) UpdateOrder(_ context.Context, _ application.Credentials, number string, body contract.OrderWrite) (contract.Order, error) {
	f.updateOrderCalls = append(f.updateOrderCalls, orderUpdateCall{Number: number, Write: body})
	order := f.updatedOrder
	if order.Number == "" {
		order.Number = number
	}
	if f.orders == nil {
		f.orders = make(map[string]contract.Order)
	}
	f.orders[order.Number] = order
	return order, nil
}
func (f *fakeBackend) CancelOrder(_ context.Context, _ application.Credentials, number string) (contract.Order, error) {
	f.cancelOrderCalls = append(f.cancelOrderCalls, number)
	if f.orders == nil {
		f.orders = make(map[string]contract.Order)
	}
	order := f.orders[number]
	order.Cancelled = true
	f.orders[number] = order
	return order, nil
}
func (f *fakeBackend) RestoreOrder(_ context.Context, _ application.Credentials, number string) (contract.Order, error) {
	f.restoreOrderCalls = append(f.restoreOrderCalls, number)
	if f.orders == nil {
		f.orders = make(map[string]contract.Order)
	}
	order := f.orders[number]
	order.Cancelled = false
	f.orders[number] = order
	return order, nil
}
func (f *fakeBackend) SetOrderFavorite(_ context.Context, _ application.Credentials, number string, favorite bool) (contract.Order, error) {
	f.favoriteCalls = append(f.favoriteCalls, favoriteCall{Number: number, Favorite: favorite})
	if f.orders == nil {
		f.orders = make(map[string]contract.Order)
	}
	order := f.orders[number]
	order.Favorite = favorite
	f.orders[number] = order
	return order, nil
}
func (f *fakeBackend) SaveOrderDraft(_ context.Context, _ application.Credentials, body contract.OrderWrite) (contract.OrderDraft, error) {
	f.savedDraftWrites = append(f.savedDraftWrites, body)
	draft := contract.OrderDraft{Write: body, UpdatedAt: "2026-08-01T00:00:00Z"}
	if f.draftByCategory == nil {
		f.draftByCategory = make(map[int64]contract.OrderDraft)
	}
	f.draftByCategory[body.CategoryID] = draft
	return draft, nil
}
func (f *fakeBackend) DeleteOrderDraft(_ context.Context, _ application.Credentials, categoryID int64) error {
	f.deletedDraftCategory = append(f.deletedDraftCategory, categoryID)
	delete(f.draftByCategory, categoryID)
	return nil
}
func (f *fakeBackend) OrderDraft(_ context.Context, _ application.Credentials, categoryID int64) (contract.OrderDraft, error) {
	if draft, ok := f.draftByCategory[categoryID]; ok {
		return draft, nil
	}
	return contract.OrderDraft{}, &application.Error{Status: http.StatusNotFound, Message: "черновик не найден"}
}
func (f *fakeBackend) OrderDrafts(context.Context, application.Credentials) ([]contract.OrderDraft, error) {
	return f.drafts, nil
}
func (f *fakeBackend) CreateProductionSheet(context.Context, application.Credentials, contract.ProductionWrite) (contract.ProductionSheet, error) {
	return contract.ProductionSheet{}, nil
}
func (f *fakeBackend) UpdateProductionSheet(context.Context, application.Credentials, int64, contract.ProductionWrite) (contract.ProductionSheet, error) {
	return contract.ProductionSheet{}, nil
}
func (f *fakeBackend) DeleteProductionSheet(context.Context, application.Credentials, int64) error {
	return nil
}
func (f *fakeBackend) CalculateDough(context.Context, application.Credentials, contract.DoughCalcRequest) ([]contract.MonitorReport, error) {
	return nil, nil
}
func (f *fakeBackend) CreateUser(context.Context, application.Credentials, contract.UserCreate) (contract.User, error) {
	return contract.User{}, nil
}
func (f *fakeBackend) UpdateUser(context.Context, application.Credentials, int64, contract.UserUpdate) (contract.User, error) {
	return contract.User{}, nil
}
func (f *fakeBackend) DeleteUser(context.Context, application.Credentials, int64) error { return nil }
func (f *fakeBackend) CreateDish(context.Context, application.Credentials, contract.DishWrite) (contract.Dish, error) {
	return contract.Dish{}, nil
}
func (f *fakeBackend) UpdateDish(context.Context, application.Credentials, string, contract.DishWrite) (contract.Dish, error) {
	return contract.Dish{}, nil
}
func (f *fakeBackend) DeleteDish(context.Context, application.Credentials, string) error { return nil }
func (f *fakeBackend) ReorderDishes(context.Context, application.Credentials, []string) error {
	return nil
}
func (f *fakeBackend) CreateCategory(context.Context, application.Credentials, contract.CategoryWrite) (contract.Category, error) {
	return contract.Category{}, nil
}
func (f *fakeBackend) UpdateCategory(context.Context, application.Credentials, int64, contract.CategoryWrite) (contract.Category, error) {
	return contract.Category{}, nil
}
func (f *fakeBackend) DeleteCategory(context.Context, application.Credentials, int64) error {
	return nil
}

func (f *fakeBackend) SyncIiko(context.Context, application.Credentials) error {
	return nil
}

// newE2E starts the real handler over httptest and returns a client whose
// cookie jar carries session + CSRF cookies between requests, and does not
// auto-follow redirects so tests can assert on 303/HX responses.
func newE2E(t *testing.T, back *fakeBackend) (*httptest.Server, *http.Client) {
	t.Helper()
	handler, err := New(back, back, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return srv, client
}

// csrfToken reads the CSRF cookie the server set on the jar for this host.
func csrfToken(t *testing.T, client *http.Client, base string) string {
	t.Helper()
	u, _ := url.Parse(base)
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == csrfCookie {
			return c.Value
		}
	}
	t.Fatal("csrf cookie not set")
	return ""
}

// postCSRF warms the CSRF cookie with a GET to warmPath, fills it into form,
// then POSTs form to path. Mirrors the warm-then-post pattern the CSRF test
// uses inline, factored out since the write-flow tests below all need it.
func postCSRF(t *testing.T, client *http.Client, base, warmPath, path string, form url.Values) *http.Response {
	t.Helper()
	warm, err := client.Get(base + warmPath)
	if err != nil {
		t.Fatalf("warm %s: %v", warmPath, err)
	}
	_ = warm.Body.Close()
	form.Set("_csrf", csrfToken(t, client, base))
	resp, err := client.PostForm(base+path, form)
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	return resp
}

func shopBackend() *fakeBackend {
	return &fakeBackend{
		loginToken: "shop-token",
		viewers: map[application.Credentials]contract.Me{
			"Bearer shop-token": {Role: "shop", TelegramUsername: "shopuser", DepartmentID: 1, DepartmentType: "shop"},
		},
	}
}

func login(t *testing.T, client *http.Client, base, username string) {
	t.Helper()
	resp, err := client.PostForm(base+"/session/login", url.Values{
		"username": {username}, "password": {"secret"}, "next": {"/orders"},
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/orders" {
		t.Fatalf("login redirect = %q, want /orders", got)
	}
}

func TestE2EUnauthenticatedOrdersShowsLogin(t *testing.T) {
	t.Parallel()
	srv, client := newE2E(t, shopBackend())
	resp, err := client.Get(srv.URL + "/orders")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Войти") {
		t.Fatal("login form not rendered for anonymous /orders")
	}
}

func TestE2ELoginThenOrders(t *testing.T) {
	t.Parallel()
	srv, client := newE2E(t, shopBackend())
	login(t, client, srv.URL, "shopuser")

	resp, err := client.Get(srv.URL + "/orders")
	if err != nil {
		t.Fatalf("get orders: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("orders status = %d, want 200", resp.StatusCode)
	}
}

func TestE2EOrdersDefaultToOneDynamicCategory(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.categories = []contract.Category{
		{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"},
		{ID: 8, Name: "Булочки", Letter: "Б", Color: "sky"},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "shopuser")

	resp, err := client.Get(srv.URL + "/orders")
	if err != nil {
		t.Fatalf("get orders: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	html := string(body)
	if back.orderFilters.CategoryID != 4 {
		t.Fatalf("category filter = %d, want first dynamic category 4", back.orderFilters.CategoryID)
	}
	if !strings.Contains(html, "Булочки") || strings.Contains(html, "Все типы") {
		t.Fatalf("dynamic category tabs were not rendered: %s", html)
	}
}

func TestE2EBakerOrdersLoadsEveryPageInTheWindow(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.viewers["Bearer shop-token"] = contract.Me{Role: "baker", TelegramUsername: "baker"}
	back.categories = []contract.Category{{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}}
	back.orderPages = map[int]contract.OrdersPage{
		1: {
			Items: []contract.Order{{
				Number:          "Г.Х.29.07.26.001",
				Category:        &back.categories[0],
				FulfillmentDate: "2026-07-29",
			}},
			Page: 1, Limit: 100, Total: 101, TotalPages: 2,
		},
		2: {
			Items: []contract.Order{{
				Number:          "Г.Х.29.07.26.101",
				Category:        &back.categories[0],
				FulfillmentDate: "2026-07-29",
			}},
			Page: 2, Limit: 100, Total: 101, TotalPages: 2,
		},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "baker")

	resp, err := client.Get(srv.URL + "/orders?category_id=4&fulfillment_from=2026-07-29&fulfillment_to=2026-08-02")
	if err != nil {
		t.Fatalf("get orders: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("orders status = %d, want 200", resp.StatusCode)
	}
	html := string(body)
	for _, number := range []string{"Г.Х.29.07.26.001", "Г.Х.29.07.26.101"} {
		if !strings.Contains(html, number) {
			t.Fatalf("orders page does not contain page item %q", number)
		}
	}
	if len(back.orderCalls) != 2 || back.orderCalls[0].Page != 1 || back.orderCalls[1].Page != 2 ||
		back.orderCalls[0].CategoryID != 4 || back.orderCalls[1].CategoryID != 4 {
		t.Fatalf("order calls = %+v, want category 4 pages 1 and 2", back.orderCalls)
	}
}

func TestE2EOrdersTableLoadsEveryPageInTheWindow(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.viewers["Bearer shop-token"] = contract.Me{Role: "baker", TelegramUsername: "baker"}
	back.categories = []contract.Category{{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}}
	back.orderPages = map[int]contract.OrdersPage{
		1: {
			Items: []contract.Order{{
				Number:          "Г.Х.29.07.26.001",
				Category:        &back.categories[0],
				FulfillmentDate: "2026-07-29",
				Items:           []contract.OrderItem{{ProductName: "Багет", ProductionQuantity: 1}},
			}},
			Page: 1, Limit: 100, Total: 101, TotalPages: 2,
		},
		2: {
			Items: []contract.Order{{
				Number:          "Г.Х.29.07.26.101",
				Category:        &back.categories[0],
				FulfillmentDate: "2026-07-29",
				Items:           []contract.OrderItem{{ProductName: "Редкая позиция 101", ProductionQuantity: 1}},
			}},
			Page: 2, Limit: 100, Total: 101, TotalPages: 2,
		},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "baker")

	resp, err := client.Get(srv.URL + "/orders/table?start=2026-07-29&category_id=4")
	if err != nil {
		t.Fatalf("get orders table: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("orders table status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Редкая позиция 101") {
		t.Fatal("orders table does not contain an item from page 2")
	}
	if len(back.orderCalls) != 2 || back.orderCalls[0].Page != 1 || back.orderCalls[1].Page != 2 ||
		back.orderCalls[0].CategoryID != 4 || back.orderCalls[1].CategoryID != 4 {
		t.Fatalf("order calls = %+v, want category 4 pages 1 and 2", back.orderCalls)
	}
}

func TestE2EOrderNewShowsSaveDraftButtonForShop(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.categories = []contract.Category{{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "shopuser")

	resp, err := client.Get(srv.URL + "/orders/new")
	if err != nil {
		t.Fatalf("get orders/new: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	html := string(body)
	if !strings.Contains(html, `formaction="/orders/draft"`) {
		t.Fatalf("shop create form should offer a save-draft button: %s", html)
	}
	if !strings.Contains(html, `href="/drafts"`) {
		t.Fatalf("shop nav should link to /drafts: %s", html)
	}
}

func TestE2EOrderNewHidesDraftUIForBaker(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.viewers["Bearer shop-token"] = contract.Me{Role: "baker", TelegramUsername: "baker"}
	back.categories = []contract.Category{{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "baker")

	resp, err := client.Get(srv.URL + "/orders/new")
	if err != nil {
		t.Fatalf("get orders/new: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	html := string(body)
	if strings.Contains(html, `formaction="/orders/draft"`) {
		t.Fatalf("baker create form must not offer a save-draft button: %s", html)
	}
	if strings.Contains(html, `href="/drafts"`) {
		t.Fatalf("baker nav must not link to /drafts: %s", html)
	}
}

func TestE2EDraftsPageListsDraftsForShop(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.categories = []contract.Category{{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}}
	back.drafts = []contract.OrderDraft{
		{Write: contract.OrderWrite{CategoryID: 4}, UpdatedAt: "2026-07-20T10:00:00Z"},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "shopuser")

	resp, err := client.Get(srv.URL + "/drafts")
	if err != nil {
		t.Fatalf("get drafts: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("drafts status = %d, want 200", resp.StatusCode)
	}
	html := string(body)
	if !strings.Contains(html, "draft-card") || !strings.Contains(html, "Хлеб") {
		t.Fatalf("drafts page should list the draft: %s", html)
	}
}

func TestE2EDraftsPageForbiddenForBaker(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.viewers["Bearer shop-token"] = contract.Me{Role: "baker", TelegramUsername: "baker"}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "baker")

	resp, err := client.Get(srv.URL + "/drafts")
	if err != nil {
		t.Fatalf("get drafts: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("drafts status = %d, want 403", resp.StatusCode)
	}
}

func TestE2EInvalidLoginRejected(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.loginErr = &application.Error{Status: http.StatusUnauthorized, Message: "неверный пароль"}
	srv, client := newE2E(t, back)

	resp, err := client.PostForm(srv.URL+"/session/login", url.Values{
		"username": {"shopuser"}, "password": {"wrong"}, "next": {"/orders"},
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if !strings.Contains(string(body), "неверный пароль") {
		t.Fatal("backend error message not surfaced on login")
	}
}

func TestE2ERBACShopBlockedFromAdminAndProduction(t *testing.T) {
	t.Parallel()
	srv, client := newE2E(t, shopBackend())
	login(t, client, srv.URL, "shopuser")

	for _, path := range []string{"/admin/users", "/production"} {
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s status = %d, want 403", path, resp.StatusCode)
		}
	}
}

func TestE2ERBACBakerReachesProductionNotAdmin(t *testing.T) {
	t.Parallel()
	back := &fakeBackend{
		loginToken: "baker-token",
		viewers: map[application.Credentials]contract.Me{
			"Bearer baker-token": {Role: "baker", TelegramUsername: "bakeruser"},
		},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "bakeruser")

	resp, err := client.Get(srv.URL + "/production")
	if err != nil {
		t.Fatalf("get production: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("production status = %d, want 200", resp.StatusCode)
	}

	resp, err = client.Get(srv.URL + "/orders/new")
	if err != nil {
		t.Fatalf("get new order: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Цех Пекари") {
		t.Fatalf("new order page = %d, workshop source missing", resp.StatusCode)
	}

	resp, err = client.Get(srv.URL + "/admin/users")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("admin status = %d, want 403", resp.StatusCode)
	}
}

func TestE2EProductionJournalOmitsSheetsWithoutCategory(t *testing.T) {
	t.Parallel()
	category := contract.Category{ID: 1, Name: "Хлеб", Letter: "Х", Color: "amber"}
	back := &fakeBackend{
		loginToken: "baker-token",
		viewers: map[application.Credentials]contract.Me{
			"Bearer baker-token": {Role: "baker", TelegramUsername: "bakeruser"},
		},
		orders: map[string]contract.Order{
			"typed":  {Number: "typed", Category: &category},
			"legacy": {Number: "legacy"},
		},
		sheets: []contract.ProductionSheet{
			{ID: 1, CreatedAt: "2026-07-24T08:00:00Z", CreatedByUsername: "bakeruser", OrderNumbers: []string{"typed"}},
			{ID: 2, CreatedAt: "2026-07-24T09:00:00Z", CreatedByUsername: "bakeruser", OrderNumbers: []string{"legacy"}},
		},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "bakeruser")

	resp, err := client.Get(srv.URL + "/production")
	if err != nil {
		t.Fatalf("get production: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	html := string(body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(html, "№1") {
		t.Fatalf("production journal = %d, categorized sheet missing", resp.StatusCode)
	}
	if strings.Contains(html, "№2") || strings.Contains(html, "Без типа") {
		t.Fatal("uncategorized production sheet reached the journal")
	}
}

func TestE2EProductionPrintPage(t *testing.T) {
	t.Parallel()
	category := contract.Category{ID: 1, Name: "Хлеб", Letter: "Х", Color: "amber"}
	back := &fakeBackend{
		loginToken: "baker-token",
		viewers: map[application.Credentials]contract.Me{
			"Bearer baker-token": {Role: "baker", TelegramUsername: "bakeruser"},
		},
		orders: map[string]contract.Order{
			"A1": {Number: "A1", Category: &category,
				FromDepartment: &contract.Department{ID: 1, Name: "Сары-Арка"},
				Items: []contract.OrderItem{
					{ProductName: "Багет", Quantity: 3, ProductionQuantity: 3},
					{ProductName: "Самса", Quantity: 5, ProductionQuantity: 5},
				}},
			"A2": {Number: "A2", Category: &category,
				FromDepartment: &contract.Department{ID: 2, Name: "Шолохова"},
				Items: []contract.OrderItem{
					{ProductName: "Багет", Quantity: 2, ProductionQuantity: 2},
				}},
		},
		sheet: contract.ProductionSheet{ID: 7, CreatedAt: "2026-08-20T08:00:00Z", OrderNumbers: []string{"A1", "A2"}},
		batchMonitor: contract.BatchMonitor{TotalReports: []contract.MonitorReport{{
			Code: "dough",
			Report: monitoringdomain.IngredientReport{
				Ingredient: monitoringdomain.IngredientUsage{ProductName: "Тесто дрожжевое", Unit: "кг", Quantity: 12.5},
				Components: []monitoringdomain.IngredientComponent{{ProductName: "Мука", Unit: "кг", Quantity: 8}},
			},
		}}},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "bakeruser")

	resp, err := client.Get(srv.URL + "/production/7/print")
	if err != nil {
		t.Fatalf("get print: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("print status = %d, want 200", resp.StatusCode)
	}
	html := string(body)
	// Shop columns, catalog group + trailing «Прочее», and the dough calc.
	for _, want := range []string{"Сары-Арка", "Шолохова", "Хлеб", "Багет", "Прочее", "Самса", "Тесто дрожжевое", "Мука"} {
		if !strings.Contains(html, want) {
			t.Errorf("print page misses %q", want)
		}
	}
	// «Багет» (catalog dish, group «Хлеб») must come before «Самса» (unknown, «Прочее»).
	if strings.Index(html, "Багет") > strings.Index(html, "Самса") {
		t.Error("print rows are not in catalog order")
	}
}

func TestE2ECreateOrderRequiresCSRF(t *testing.T) {
	t.Parallel()
	srv, client := newE2E(t, shopBackend())
	login(t, client, srv.URL, "shopuser")

	form := url.Values{
		"category_id": {"1"}, "fulfillment_date": {"2026-08-01"},
		"product_name": {"Багет"}, "quantity": {"3"},
	}

	// Without a CSRF token the middleware rejects the write.
	resp, err := client.PostForm(srv.URL+"/orders", form)
	if err != nil {
		t.Fatalf("post no-csrf: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("no-csrf status = %d, want 403", resp.StatusCode)
	}

	// GET a page so the server issues the CSRF cookie, then replay with it.
	warm, err := client.Get(srv.URL + "/orders/new")
	if err != nil {
		t.Fatalf("warm: %v", err)
	}
	_ = warm.Body.Close()
	form.Set("_csrf", csrfToken(t, client, srv.URL))

	resp, err = client.PostForm(srv.URL+"/orders", form)
	if err != nil {
		t.Fatalf("post csrf: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("valid CSRF still rejected: %d", resp.StatusCode)
	}
}

func TestE2ECreateOrderSuccess(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	category := contract.Category{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}
	back.categories = []contract.Category{category}
	back.createdOrder = contract.Order{
		Number:          "Г.Х.01.08.26.001",
		Category:        &category,
		FromDepartment:  &contract.Department{ID: 1, Name: "Магазин 1"},
		FulfillmentDate: "2026-08-01",
		Items:           []contract.OrderItem{{ProductName: "Багет", Quantity: 3}},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "shopuser")

	form := url.Values{
		"category_id": {"4"}, "from_department_id": {"1"}, "fulfillment_date": {"2026-08-01"},
		"item_name": {"Багет"}, "quantity": {"3"},
	}
	resp := postCSRF(t, client, srv.URL, "/orders/new", "/orders", form)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303", resp.StatusCode)
	}
	wantLocation := "/orders/" + url.PathEscape("Г.Х.01.08.26.001") + "?success=" + url.QueryEscape("Заказ создан.")
	if got := resp.Header.Get("Location"); got != wantLocation {
		t.Fatalf("create redirect = %q, want %q", got, wantLocation)
	}

	if len(back.createOrderCalls) != 1 {
		t.Fatalf("CreateOrder calls = %d, want 1", len(back.createOrderCalls))
	}
	call := back.createOrderCalls[0]
	if call.CategoryID != 4 || call.FulfillmentDate != "2026-08-01" {
		t.Fatalf("CreateOrder body = %+v, want category 4, date 2026-08-01", call)
	}
	if call.FromDepartmentID == nil || *call.FromDepartmentID != 1 {
		t.Fatalf("CreateOrder from_department_id = %v, want 1", call.FromDepartmentID)
	}
	if len(call.Items) != 1 || call.Items[0].ProductName != "Багет" || call.Items[0].Quantity != 3 {
		t.Fatalf("CreateOrder items = %+v, want one Багет x3", call.Items)
	}

	page, err := client.Get(srv.URL + wantLocation)
	if err != nil {
		t.Fatalf("get created order: %v", err)
	}
	body, _ := io.ReadAll(page.Body)
	_ = page.Body.Close()
	html := string(body)
	if page.StatusCode != http.StatusOK {
		t.Fatalf("created order page status = %d, want 200", page.StatusCode)
	}
	if !strings.Contains(html, "Г.Х.01.08.26.001") || !strings.Contains(html, "Багет") || !strings.Contains(html, "Заказ создан.") {
		t.Fatalf("created order page missing expected content: %s", html)
	}
}

func TestE2EUpdateOrderSuccess(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	category := contract.Category{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}
	back.categories = []contract.Category{category}
	const number = "Г.Х.01.08.26.001"
	back.orders = map[string]contract.Order{
		number: {
			Number: number, Category: &category,
			FromDepartment:  &contract.Department{ID: 1, Name: "Магазин 1"},
			FulfillmentDate: "2026-08-01", Items: []contract.OrderItem{{ProductName: "Багет", Quantity: 2}},
		},
	}
	back.updatedOrder = contract.Order{
		Number: number, Category: &category,
		FromDepartment:  &contract.Department{ID: 1, Name: "Магазин 1"},
		FulfillmentDate: "2026-08-02",
		Items:           []contract.OrderItem{{ProductName: "Багет", Quantity: 5}},
		Comments:        contract.Comments{General: "обновлено"},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "shopuser")
	editPath := "/orders/" + url.PathEscape(number) + "/edit"

	form := url.Values{
		"from_department_id": {"1"}, "fulfillment_date": {"2026-08-02"},
		"item_name": {"Багет"}, "quantity": {"5"}, "general_comment": {"обновлено"},
	}
	resp := postCSRF(t, client, srv.URL, editPath, editPath, form)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("update status = %d, want 303", resp.StatusCode)
	}
	wantLocation := "/orders/" + url.PathEscape(number) + "?success=" + url.QueryEscape("Изменения сохранены.")
	if got := resp.Header.Get("Location"); got != wantLocation {
		t.Fatalf("update redirect = %q, want %q", got, wantLocation)
	}

	if len(back.updateOrderCalls) != 1 {
		t.Fatalf("UpdateOrder calls = %d, want 1", len(back.updateOrderCalls))
	}
	call := back.updateOrderCalls[0]
	if call.Number != number {
		t.Fatalf("UpdateOrder number = %q, want %s", call.Number, number)
	}
	if call.Write.FulfillmentDate != "2026-08-02" || call.Write.Comments.General != "обновлено" {
		t.Fatalf("UpdateOrder body = %+v, want date 2026-08-02, comment обновлено", call.Write)
	}
	if len(call.Write.Items) != 1 || call.Write.Items[0].Quantity != 5 {
		t.Fatalf("UpdateOrder items = %+v, want one item x5", call.Write.Items)
	}

	page, err := client.Get(srv.URL + wantLocation)
	if err != nil {
		t.Fatalf("get updated order: %v", err)
	}
	body, _ := io.ReadAll(page.Body)
	_ = page.Body.Close()
	html := string(body)
	if !strings.Contains(html, "обновлено") || !strings.Contains(html, "Изменения сохранены.") {
		t.Fatalf("updated order page does not reflect the change: %s", html)
	}
}

func TestE2ECancelAndRestoreOrder(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	category := contract.Category{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}
	back.categories = []contract.Category{category}
	const number = "Г.Х.01.08.26.001"
	back.orders = map[string]contract.Order{
		number: {Number: number, Category: &category},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "shopuser")
	orderPath := "/orders/" + url.PathEscape(number)

	cancelResp := postCSRF(t, client, srv.URL, orderPath, orderPath+"/cancel", url.Values{})
	_ = cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("cancel status = %d, want 303", cancelResp.StatusCode)
	}
	if len(back.cancelOrderCalls) != 1 || back.cancelOrderCalls[0] != number {
		t.Fatalf("CancelOrder calls = %v, want one call for %s", back.cancelOrderCalls, number)
	}

	afterCancel, err := client.Get(srv.URL + orderPath)
	if err != nil {
		t.Fatalf("get after cancel: %v", err)
	}
	body, _ := io.ReadAll(afterCancel.Body)
	_ = afterCancel.Body.Close()
	if !strings.Contains(string(body), "Отменён") || !strings.Contains(string(body), "Восстановить заказ") {
		t.Fatalf("order page does not reflect cancellation: %s", body)
	}

	restoreResp := postCSRF(t, client, srv.URL, orderPath, orderPath+"/restore", url.Values{})
	_ = restoreResp.Body.Close()
	if restoreResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("restore status = %d, want 303", restoreResp.StatusCode)
	}
	if len(back.restoreOrderCalls) != 1 || back.restoreOrderCalls[0] != number {
		t.Fatalf("RestoreOrder calls = %v, want one call for %s", back.restoreOrderCalls, number)
	}

	afterRestore, err := client.Get(srv.URL + orderPath)
	if err != nil {
		t.Fatalf("get after restore: %v", err)
	}
	body, _ = io.ReadAll(afterRestore.Body)
	_ = afterRestore.Body.Close()
	if strings.Contains(string(body), "Отменён") || !strings.Contains(string(body), "Отменить заказ") {
		t.Fatalf("order page does not reflect restoration: %s", body)
	}
}

func TestE2ESetOrderFavorite(t *testing.T) {
	t.Parallel()
	back := &fakeBackend{
		loginToken: "admin-token",
		viewers: map[application.Credentials]contract.Me{
			"Bearer admin-token": {Role: "admin", TelegramUsername: "adminuser"},
		},
	}
	category := contract.Category{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}
	back.categories = []contract.Category{category}
	const number = "Г.Х.01.08.26.001"
	back.orders = map[string]contract.Order{
		number: {Number: number, Category: &category, Favorite: false},
	}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "adminuser")
	orderPath := "/orders/" + url.PathEscape(number)

	resp := postCSRF(t, client, srv.URL, orderPath, orderPath+"/favorite", url.Values{"favorite": {"true"}})
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("favorite status = %d, want 303", resp.StatusCode)
	}
	if len(back.favoriteCalls) != 1 || back.favoriteCalls[0] != (favoriteCall{Number: number, Favorite: true}) {
		t.Fatalf("SetOrderFavorite calls = %+v, want one true call for %s", back.favoriteCalls, number)
	}

	page, err := client.Get(srv.URL + orderPath)
	if err != nil {
		t.Fatalf("get after favorite: %v", err)
	}
	body, _ := io.ReadAll(page.Body)
	_ = page.Body.Close()
	if !strings.Contains(string(body), "Убрать из избранного") {
		t.Fatalf("order page does not reflect favorite toggle: %s", body)
	}
}

func TestE2EDraftSaveLoadDeleteRoundTrip(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	category := contract.Category{ID: 4, Name: "Хлеб", Letter: "Х", Color: "amber"}
	back.categories = []contract.Category{category}
	srv, client := newE2E(t, back)
	login(t, client, srv.URL, "shopuser")

	saveForm := url.Values{
		"category_id": {"4"}, "from_department_id": {"1"}, "fulfillment_date": {"2026-08-03"},
		"item_name": {"Багет"}, "quantity": {"2"},
	}
	saveResp := postCSRF(t, client, srv.URL, "/orders/new", "/orders/draft", saveForm)
	_ = saveResp.Body.Close()
	if saveResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("save draft status = %d, want 303", saveResp.StatusCode)
	}
	wantSaveLocation := "/orders/new?draft_category_id=4&success=" + url.QueryEscape("Черновик сохранён.")
	if got := saveResp.Header.Get("Location"); got != wantSaveLocation {
		t.Fatalf("save draft redirect = %q, want %q", got, wantSaveLocation)
	}
	if len(back.savedDraftWrites) != 1 || back.savedDraftWrites[0].CategoryID != 4 || back.savedDraftWrites[0].FulfillmentDate != "2026-08-03" {
		t.Fatalf("SaveOrderDraft calls = %+v, want one draft for category 4 on 2026-08-03", back.savedDraftWrites)
	}

	loaded, err := client.Get(srv.URL + "/orders/new?draft_category_id=4")
	if err != nil {
		t.Fatalf("get draft-loaded form: %v", err)
	}
	body, _ := io.ReadAll(loaded.Body)
	_ = loaded.Body.Close()
	if !strings.Contains(string(body), `value="2"`) || !strings.Contains(string(body), `value="2026-08-03"`) {
		t.Fatalf("draft-loaded form does not prefill the saved values: %s", body)
	}

	deleteResp := postCSRF(t, client, srv.URL, "/orders/new", "/orders/draft/4/delete", url.Values{})
	_ = deleteResp.Body.Close()
	if deleteResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete draft status = %d, want 303", deleteResp.StatusCode)
	}
	wantDeleteLocation := "/drafts?success=" + url.QueryEscape("Черновик удалён.")
	if got := deleteResp.Header.Get("Location"); got != wantDeleteLocation {
		t.Fatalf("delete draft redirect = %q, want %q", got, wantDeleteLocation)
	}
	if len(back.deletedDraftCategory) != 1 || back.deletedDraftCategory[0] != 4 {
		t.Fatalf("DeleteOrderDraft calls = %v, want one call for category 4", back.deletedDraftCategory)
	}

	afterDelete, err := client.Get(srv.URL + "/orders/new?draft_category_id=4")
	if err != nil {
		t.Fatalf("get after draft delete: %v", err)
	}
	body, _ = io.ReadAll(afterDelete.Body)
	_ = afterDelete.Body.Close()
	if strings.Contains(string(body), `value="2"`) {
		t.Fatalf("deleted draft still prefills the form: %s", body)
	}
}

func TestE2ELogoutClearsSession(t *testing.T) {
	t.Parallel()
	srv, client := newE2E(t, shopBackend())
	login(t, client, srv.URL, "shopuser")
	form := url.Values{"_csrf": {""}}

	warm, _ := client.Get(srv.URL + "/orders")
	_ = warm.Body.Close()
	form.Set("_csrf", csrfToken(t, client, srv.URL))

	resp, err := client.PostForm(srv.URL+"/session/logout", form)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want 303", resp.StatusCode)
	}

	// Session gone: /orders falls back to the login page.
	after, err := client.Get(srv.URL + "/orders")
	if err != nil {
		t.Fatalf("get after logout: %v", err)
	}
	_ = after.Body.Close()
	if after.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout /orders = %d, want 401", after.StatusCode)
	}
}

func TestE2EPasswordFallbackInMiniAppSetsTmaSession(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.viewers["tma bound-init-data"] = contract.Me{Role: "shop", TelegramUsername: "shopuser", DepartmentID: 1, DepartmentType: "shop"}
	srv, client := newE2E(t, back)

	resp, err := client.PostForm(srv.URL+"/session/login", url.Values{
		"username": {"shopuser"}, "password": {"secret"}, "init_data": {"bound-init-data"}, "next": {"/orders"},
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status = %d, want 303", resp.StatusCode)
	}

	// The session must run on tma credentials, not the bearer token: logout is
	// refused exactly for tma sessions.
	warm, _ := client.Get(srv.URL + "/orders")
	_ = warm.Body.Close()
	form := url.Values{"_csrf": {csrfToken(t, client, srv.URL)}}
	resp, err = client.PostForm(srv.URL+"/session/logout", form)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	_ = resp.Body.Close()
	if resp.Header.Get("Location") != "/me" {
		t.Fatalf("logout redirect = %q, want /me (tma session)", resp.Header.Get("Location"))
	}
}

func TestE2EPasswordFallbackWithUnboundTelegramFailsWithoutBearerBypass(t *testing.T) {
	t.Parallel()
	// Password is right, but the tma credential resolves no user (bind did not
	// take effect) — the login must fail instead of falling back to a bearer
	// session inside the mini app.
	srv, client := newE2E(t, shopBackend())

	resp, err := client.PostForm(srv.URL+"/session/login", url.Values{
		"username": {"shopuser"}, "password": {"secret"}, "init_data": {"unbound-init-data"}, "next": {"/orders"},
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusSeeOther {
		t.Fatal("login succeeded, want failure page without a session")
	}
	after, err := client.Get(srv.URL + "/orders")
	if err != nil {
		t.Fatalf("get /orders: %v", err)
	}
	_ = after.Body.Close()
	if after.StatusCode == http.StatusOK {
		t.Fatal("bearer session leaked into the mini app flow")
	}
}

func TestE2EMiniAppSessionCannotLogout(t *testing.T) {
	t.Parallel()
	back := shopBackend()
	back.viewers["tma test-init-data"] = contract.Me{Role: "shop", TelegramUsername: "shopuser", DepartmentID: 1, DepartmentType: "shop"}
	srv, client := newE2E(t, back)

	resp, err := client.PostForm(srv.URL+"/session/telegram", url.Values{"init_data": {"test-init-data"}, "next": {"/orders"}})
	if err != nil {
		t.Fatalf("telegram login: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("telegram login status = %d, want 204", resp.StatusCode)
	}

	// /me hides the logout button for a tma session.
	me, err := client.Get(srv.URL + "/me")
	if err != nil {
		t.Fatalf("get /me: %v", err)
	}
	body, _ := io.ReadAll(me.Body)
	_ = me.Body.Close()
	if strings.Contains(string(body), "Выйти") {
		t.Fatal("/me shows logout button for mini app session")
	}

	// A direct logout POST must not clear the session.
	form := url.Values{"_csrf": {csrfToken(t, client, srv.URL)}}
	resp, err = client.PostForm(srv.URL+"/session/logout", form)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/me" {
		t.Fatalf("logout = %d %q, want 303 /me", resp.StatusCode, resp.Header.Get("Location"))
	}
	after, err := client.Get(srv.URL + "/orders")
	if err != nil {
		t.Fatalf("get after logout: %v", err)
	}
	_ = after.Body.Close()
	if after.StatusCode != http.StatusOK {
		t.Fatalf("post-logout /orders = %d, want 200 (session must survive)", after.StatusCode)
	}
}

func TestE2EHealthOK(t *testing.T) {
	t.Parallel()
	srv, client := newE2E(t, shopBackend())
	resp, err := client.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"status":"ok"`) {
		t.Fatalf("health = %d %s", resp.StatusCode, body)
	}
}
