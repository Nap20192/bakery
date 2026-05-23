package iiko

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		_ = godotenv.Load()
		return
	}
	_ = godotenv.Load(filepath.Join(filepath.Dir(file), "../../../.env"))
}

func TestClient_AllMethods(t *testing.T) {
	resultsDir := testResultsDir(t)

	requireEnv := func(name string) string {
		t.Helper()
		v := os.Getenv(name)
		if v == "" {
			t.Skipf("%s is not set", name)
		}
		return v
	}

	iikoHost := requireEnv("IIKO_HOST")
	iikoPort := requireEnv("IIKO_PORT")
	iikoLogin := requireEnv("IIKO_LOGIN")
	iikoPassword := requireEnv("IIKO_PASSWORD")
	depID := os.Getenv("IIKO_DEPARTMENT_ID")

	api := NewApi(iikoHost, iikoPort)
	client, err := NewClient(iikoLogin, iikoPassword, api)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if err := client.Auth(); err != nil {
		writeJSON(t, resultsDir, "01_auth.json", map[string]string{"error": err.Error()})
		t.Fatalf("Auth failed: %v", err)
	}
	writeJSON(t, resultsDir, "01_auth.json", map[string]string{"status": "ok"})
	t.Log("Auth: OK")

	defer func() {
		_ = client.Logout()
	}()

	products, err := client.ListProducts()
	if err != nil {
		writeJSON(t, resultsDir, "02_list_products.json", map[string]string{"error": err.Error()})
		t.Fatalf("ListProducts failed: %v", err)
	}
	writeJSON(t, resultsDir, "02_list_products.json", products)
	if len(products) == 0 {
		t.Fatal("No products found, cannot continue specific tests")
	}

	t.Logf("ListProducts: OK (found %d products)", len(products))

	testID := products[0].ID.String()
	today := time.Now().Format("2006-01-02")

	resAll, err := client.AssemblyChartsGetAll(today, today, false, true)
	if err != nil {
		writeJSON(t, resultsDir, "03_assembly_charts_get_all.json", map[string]string{"error": err.Error()})
		t.Errorf("AssemblyChartsGetAll failed: %v", err)
	} else {
		writeJSON(t, resultsDir, "03_assembly_charts_get_all.json", resAll)
		t.Log("AssemblyChartsGetAll: OK")
	}

	if resAll != nil && len(resAll.AssemblyCharts) > 0 {
		resByID, err := client.AssemblyChartByID(resAll.AssemblyCharts[0].ID)
		if err != nil {
			writeJSON(t, resultsDir, "05_assembly_chart_by_id.json", map[string]string{"error": err.Error()})
			t.Errorf("AssemblyChartByID failed: %v", err)
		} else {
			writeJSON(t, resultsDir, "05_assembly_chart_by_id.json", resByID)
			t.Log("AssemblyChartByID: OK")
		}
	}

	resAssembled, err := client.AssemblyChartsGetAssembled(today, testID, depID)
	if err != nil {
		writeJSON(t, resultsDir, "07_assembly_charts_get_assembled.json", map[string]string{"error": err.Error()})
		t.Errorf("AssemblyChartsGetAssembled failed: %v", err)
	} else {
		writeJSON(t, resultsDir, "07_assembly_charts_get_assembled.json", resAssembled)
		t.Log("AssemblyChartsGetAssembled: OK")
	}

	resPrepared, err := client.AssemblyChartsGetPrepared(today, testID, depID)
	if err != nil {
		writeJSON(t, resultsDir, "08_assembly_charts_get_prepared.json", map[string]string{"error": err.Error()})
		t.Errorf("AssemblyChartsGetPrepared failed: %v", err)
	} else {
		writeJSON(t, resultsDir, "08_assembly_charts_get_prepared.json", resPrepared)
		t.Log("AssemblyChartsGetPrepared: OK")
	}
}

func testResultsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	dir := filepath.Join(filepath.Dir(file), "testdata", "results")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("create results dir: %v", err)
	}
	return dir
}

func writeJSON(t *testing.T, dir, name string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil { //nolint:gosec // test helper writes fixed fixture names.
		t.Fatalf("write %s: %v", name, err)
	}
}
