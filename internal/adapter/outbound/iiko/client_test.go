package iiko

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"
)

func newLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func pretty(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func today() string {
	return time.Now().Format("2006-01-02")
}

func testConfig(t *testing.T) (host, port, login, password string) {
	t.Helper()
	host = os.Getenv("IIKO_HOST")
	port = os.Getenv("IIKO_PORT")
	login = os.Getenv("IIKO_LOGIN")
	password = os.Getenv("IIKO_PASSWORD")
	if host == "" || login == "" || password == "" {
		t.Skip("IIKO_HOST / IIKO_LOGIN / IIKO_PASSWORD not set — skipping integration test")
	}
	if port == "" {
		port = "443"
	}
	return
}

func mustAuth(t *testing.T, log *slog.Logger) *Client {
	t.Helper()

	host, port, login, password := testConfig(t)
	api := NewApi(host, port)
	client, err := NewClient(login, password, api)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.Auth(); err != nil {
		t.Fatalf("Auth: %v", err)
	}
	log.Info("authenticated", "key", client.key)

	t.Cleanup(func() {
		if err := client.Logout(); err != nil {
			log.Error("logout failed", "err", err)
			return
		}
		log.Info("logged out — license slot released")
	})

	return client
}

func TestAuth(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestAuth ===")

	host, port, login, password := testConfig(t)
	api := NewApi(host, port)
	client, err := NewClient(login, password, api)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.Auth(); err != nil {
		t.Fatalf("Auth failed: %v", err)
	}

	if client.key == "" {
		t.Fatal("expected non-empty token")
	}

	if err := client.Logout(); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	log.Info("logout ok")
}

// ─── Products ──────────────────────────────────────────────────────────────

func TestListProducts(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestListProducts ===")

	client := mustAuth(t, log)

	products, err := client.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}

	log.Info("products received", "count", len(products))
	if len(products) == 0 {
		t.Log("WARN: products list is empty")
		return
	}

	// Логируем первые 3 для наглядности
	limit := 3
	if len(products) < limit {
		limit = len(products)
	}
	for i, p := range products[:limit] {
		log.Debug("product", "index", i, "data", pretty(p))
	}
}

// ─── AssemblyCharts / getAll ────────────────────────────────────────────────

func TestAssemblyChartsGetAll(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestAssemblyChartsGetAll ===")

	client := mustAuth(t, log)

	dateFrom := "2024-01-01"
	dateTo := today()

	result, err := client.AssemblyChartsGetAll(dateFrom, dateTo, true, false)
	if err != nil {
		t.Fatalf("AssemblyChartsGetAll: %v", err)
	}

	log.Info("getAll response",
		"knownRevision", result.KnownRevision,
		"assemblyCharts", len(result.AssemblyCharts),
		"preparedCharts", len(result.PreparedCharts),
	)

	if len(result.AssemblyCharts) > 0 {
		log.Debug("first assemblyChart", "data", pretty(result.AssemblyCharts[0]))
	}
}

// ─── AssemblyCharts / getAllUpdate ──────────────────────────────────────────

func TestAssemblyChartsGetAllUpdate(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestAssemblyChartsGetAllUpdate ===")

	client := mustAuth(t, log)

	// Сначала получаем knownRevision
	dateFrom := "2024-01-01"
	dateTo := today()

	first, err := client.AssemblyChartsGetAll(dateFrom, dateTo, true, false)
	if err != nil {
		t.Fatalf("initial getAll: %v", err)
	}
	log.Info("initial revision", "knownRevision", first.KnownRevision)

	// Теперь дельта
	update, err := client.AssemblyChartsGetAllUpdate(first.KnownRevision, dateFrom, dateTo, true, false)
	if err != nil {
		t.Fatalf("AssemblyChartsGetAllUpdate: %v", err)
	}

	log.Info("getAllUpdate response",
		"knownRevision", update.KnownRevision,
		"newOrChanged", len(update.AssemblyCharts),
		"deleted", len(update.DeletedAssemblyChartIDs),
	)
	log.Debug("getAllUpdate full", "data", pretty(update))
}

// ─── AssemblyCharts / byId ──────────────────────────────────────────────────

func TestAssemblyChartByID(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestAssemblyChartByID ===")

	client := mustAuth(t, log)

	// Берём id первой доступной техкарты
	all, err := client.AssemblyChartsGetAll("2024-01-01", today(), true, false)
	if err != nil {
		t.Fatalf("getAll for id: %v", err)
	}
	if len(all.AssemblyCharts) == 0 {
		t.Skip("no assembly charts available — skipping byId test")
	}

	id := all.AssemblyCharts[0].ID
	log.Info("fetching by id", "id", id)

	result, err := client.AssemblyChartByID(id)
	if err != nil {
		t.Fatalf("AssemblyChartByID: %v", err)
	}

	log.Info("byId response", "charts", len(result.AssemblyCharts))
	log.Debug("byId full", "data", pretty(result))
}

// ─── AssemblyCharts / getTree ───────────────────────────────────────────────

func TestAssemblyChartsGetTree(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestAssemblyChartsGetTree ===")

	client := mustAuth(t, log)

	all, err := client.AssemblyChartsGetAll("2024-01-01", today(), true, false)
	if err != nil {
		t.Fatalf("getAll for tree: %v", err)
	}
	if len(all.AssemblyCharts) == 0 {
		t.Skip("no assembly charts — skipping getTree")
	}

	chart := all.AssemblyCharts[0]
	log.Info("getTree request", "productId", chart.AssembledProductID, "date", today())

	result, err := client.AssemblyChartsGetTree(today(), chart.AssembledProductID, "")
	if err != nil {
		t.Fatalf("AssemblyChartsGetTree: %v", err)
	}

	log.Info("getTree response",
		"assemblyCharts", len(result.AssemblyCharts),
		"preparedCharts", len(result.PreparedCharts),
	)
	log.Debug("getTree full", "data", pretty(result))
}

// ─── AssemblyCharts / getAssembled ─────────────────────────────────────────

func TestAssemblyChartsGetAssembled(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestAssemblyChartsGetAssembled ===")

	client := mustAuth(t, log)

	all, err := client.AssemblyChartsGetAll("2024-01-01", today(), true, false)
	if err != nil {
		t.Fatalf("getAll for assembled: %v", err)
	}
	if len(all.AssemblyCharts) == 0 {
		t.Skip("no assembly charts — skipping getAssembled")
	}

	productID := all.AssemblyCharts[0].AssembledProductID
	log.Info("getAssembled request", "productId", productID, "date", today())

	result, err := client.AssemblyChartsGetAssembled(today(), productID, "")
	if err != nil {
		t.Fatalf("AssemblyChartsGetAssembled: %v", err)
	}

	log.Info("getAssembled response", "charts", len(result.AssemblyCharts))
	log.Debug("getAssembled full", "data", pretty(result))
}

// ─── AssemblyCharts / getPrepared ──────────────────────────────────────────

func TestAssemblyChartsGetPrepared(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestAssemblyChartsGetPrepared ===")

	client := mustAuth(t, log)

	all, err := client.AssemblyChartsGetAll("2024-01-01", today(), true, false)
	if err != nil {
		t.Fatalf("getAll for prepared: %v", err)
	}
	if len(all.AssemblyCharts) == 0 {
		t.Skip("no assembly charts — skipping getPrepared")
	}

	productID := all.AssemblyCharts[0].AssembledProductID
	log.Info("getPrepared request", "productId", productID, "date", today())

	result, err := client.AssemblyChartsGetPrepared(today(), productID, "")
	if err != nil {
		t.Fatalf("AssemblyChartsGetPrepared: %v", err)
	}

	log.Info("getPrepared response", "preparedCharts", len(result.PreparedCharts))
	if len(result.PreparedCharts) > 0 {
		log.Debug("first preparedChart", "data", pretty(result.PreparedCharts[0]))
	}
}

// ─── AssemblyCharts / getHistory ───────────────────────────────────────────

func TestAssemblyChartsGetHistory(t *testing.T) {
	log := newLogger(t)
	log.Info("=== TestAssemblyChartsGetHistory ===")

	client := mustAuth(t, log)

	all, err := client.AssemblyChartsGetAll("2024-01-01", today(), true, false)
	if err != nil {
		t.Fatalf("getAll for history: %v", err)
	}
	if len(all.AssemblyCharts) == 0 {
		t.Skip("no assembly charts — skipping getHistory")
	}

	productID := all.AssemblyCharts[0].AssembledProductID
	log.Info("getHistory request", "productId", productID)

	result, err := client.AssemblyChartsGetHistory(productID, "")
	if err != nil {
		t.Fatalf("AssemblyChartsGetHistory: %v", err)
	}

	log.Info("getHistory response", "charts", len(result.AssemblyCharts))
	log.Debug("getHistory full", "data", pretty(result))
}
