package iiko

import (
	"os"
	"testing"
	"time"
)

func TestClient_AllMethods(t *testing.T) {
	if os.Getenv("IIKO_INTEGRATION") == "" {
		t.Skip("set IIKO_INTEGRATION=1 to run iiko integration tests")
	}

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

	// 1. Auth
	if err := client.Auth(); err != nil {
		t.Fatalf("Auth failed: %v", err)
	}
	t.Log("Auth: OK")

	// Гарантированный Logout в конце
	defer func() {
		_ = client.Logout()
	}()

	// 2. ListProducts (нужен для получения ID реального продукта)
	products, err := client.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts failed: %v", err)
	}
	if len(products) == 0 {
		t.Fatal("No products found, cannot continue specific tests")
	}

	t.Logf("ListProducts: OK (found %d products)", len(products))

	// Выбираем первый попавшийся ID продукта и дату для тестово
	testID := products[0].ID.String()
	today := time.Now().Format("2006-01-02")

	// 3. AssemblyChartsGetAll
	resAll, err := client.AssemblyChartsGetAll(today, today, false, true)
	if err != nil {
		t.Errorf("AssemblyChartsGetAll failed: %v", err)
	} else {
		t.Log("AssemblyChartsGetAll: OK")
	}

	// 4. AssemblyChartsGetAllUpdate
	knownRev := 0
	if resAll != nil {
		knownRev = resAll.KnownRevision
		if knownRev < 0 {
			knownRev = 0
		}
	}
	_, err = client.AssemblyChartsGetAllUpdate(knownRev, today, today, false, true)
	if err != nil {
		t.Errorf("AssemblyChartsGetAllUpdate failed: %v", err)
	} else {
		t.Log("AssemblyChartsGetAllUpdate: OK")
	}

	// 5. AssemblyChartByID (если есть данные в resAll)
	if resAll != nil && len(resAll.AssemblyCharts) > 0 {
		_, err = client.AssemblyChartByID(resAll.AssemblyCharts[0].ID)
		if err != nil {
			t.Errorf("AssemblyChartByID failed: %v", err)
		} else {
			t.Log("AssemblyChartByID: OK")
		}
	}

	// 6. AssemblyChartsGetTree
	_, err = client.AssemblyChartsGetTree(today, testID, depID)
	if err != nil {
		t.Errorf("AssemblyChartsGetTree failed: %v", err)
	} else {
		t.Log("AssemblyChartsGetTree: OK")
	}

	// 7. AssemblyChartsGetAssembled
	_, err = client.AssemblyChartsGetAssembled(today, testID, depID)
	if err != nil {
		t.Errorf("AssemblyChartsGetAssembled failed: %v", err)
	} else {
		t.Log("AssemblyChartsGetAssembled: OK")
	}

	// 8. AssemblyChartsGetPrepared
	_, err = client.AssemblyChartsGetPrepared(today, testID, depID)
	if err != nil {
		t.Errorf("AssemblyChartsGetPrepared failed: %v", err)
	} else {
		t.Log("AssemblyChartsGetPrepared: OK")
	}

	// 9. AssemblyChartsGetHistory
	_, err = client.AssemblyChartsGetHistory(testID, depID)
	if err != nil {
		t.Errorf("AssemblyChartsGetHistory failed: %v", err)
	} else {
		t.Log("AssemblyChartsGetHistory: OK")
	}

	// Logout вызовется через defer.
}
