package web

import (
	"bytes"
	"math"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"bakery/internal/inbound/api/contract"
)

func TestProductionBatchUsesProductTotals(t *testing.T) {
	t.Parallel()
	orders := []contract.Order{
		{
			Number: "A",
			Items: []contract.OrderItem{
				{ProductName: "Багет", ProductionQuantity: 10},
				{ProductName: "Батон", ProductionQuantity: 4},
			},
		},
		{
			Number: "B",
			Items:  []contract.OrderItem{{ProductName: "Багет", ProductionQuantity: 20}},
		},
	}

	rows := buildProductionRows(orders, nil)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 aggregated products", len(rows))
	}
	baguette := rows[0]
	if baguette.ProductName != "Багет" || baguette.OrderedQuantity != 30 || len(baguette.Shares) != 2 {
		t.Fatalf("baguette row = %#v", baguette)
	}
	saved := []contract.ProductionSheetItem{
		{OrderNumber: "A", ProductName: "Багет", LoadedQuantity: 9, ProducedQuantity: 8, Reason: "Подгорело"},
		{OrderNumber: "B", ProductName: "Багет", LoadedQuantity: 18, ProducedQuantity: 16, Reason: "Подгорело"},
	}
	savedBaguette := buildProductionRows(orders, saved)[0]
	if savedBaguette.LoadedQuantity != 27 || savedBaguette.ProducedQuantity != 24 || savedBaguette.Reason != "Подгорело" {
		t.Fatalf("saved baguette row = %#v", savedBaguette)
	}
	saved[1].Reason = "Упало"
	if conflict := buildProductionRows(orders, saved)[0]; !conflict.ReasonConflict || conflict.Reason != "" {
		t.Fatalf("conflicting reasons were flattened silently: %#v", conflict)
	}

	request := httptest.NewRequest("POST", "/production", nil)
	request.Form = url.Values{
		"product_name":      {"Багет"},
		"loaded_quantity":   {"27"},
		"produced_quantity": {"24"},
		"reason":            {"Подгорело"},
		"share_count":       {"2"},
		"order_number":      {"A", "B"},
		"ordered_quantity":  {"10", "20"},
	}
	body, err := parseProductionWrite(request)
	if err != nil {
		t.Fatalf("parseProductionWrite: %v", err)
	}
	if len(body.Orders) != 2 {
		t.Fatalf("orders = %d, want 2", len(body.Orders))
	}
	assertProductionItem(t, body.Orders[0], "A", 9, 8)
	assertProductionItem(t, body.Orders[1], "B", 18, 16)
}

func TestBuildProductionRowsTracksLoadedOutputLink(t *testing.T) {
	t.Parallel()
	order := contract.Order{
		Number: "A",
		Items:  []contract.OrderItem{{ProductName: "Багет", ProductionQuantity: 10}},
	}
	tests := []struct {
		name     string
		loaded   float64
		produced float64
		want     bool
	}{
		{name: "saved output follows saved load", loaded: 8, produced: 8, want: true},
		{name: "legacy output differs from load", loaded: 8, produced: 7, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := []contract.ProductionSheetItem{{
				OrderNumber:      "A",
				ProductName:      "Багет",
				LoadedQuantity:   tt.loaded,
				ProducedQuantity: tt.produced,
			}}
			rows := buildProductionRows([]contract.Order{order}, saved)
			if len(rows) != 1 || rows[0].Linked != tt.want {
				t.Fatalf("rows = %#v, linked want %t", rows, tt.want)
			}
		})
	}
}

func TestDistributeProductionQuantity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		total   float64
		ordered []float64
		want    []float64
	}{
		{name: "thirds keep the full total", total: 1, ordered: []float64{1, 1, 1}, want: []float64{0.3, 0.3, 0.4}},
		{name: "more shares than tenths stay nonnegative", total: 1, ordered: []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
		{name: "zero total", total: 0, ordered: []float64{1, 2, 3}, want: []float64{0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := distributeProductionQuantity(tt.total, tt.ordered)
			if len(got) != len(tt.ordered) {
				t.Fatalf("shares = %d, want %d", len(got), len(tt.ordered))
			}
			var totalTenths int64
			for index, quantity := range got {
				if quantity < 0 {
					t.Fatalf("share %d = %v, want nonnegative distribution %v", index, quantity, got)
				}
				totalTenths += int64(math.Round(quantity * 10))
			}
			if want := int64(math.Round(tt.total * 10)); totalTenths != want {
				t.Fatalf("distribution = %v, total tenths = %d, want %d", got, totalTenths, want)
			}
			if tt.want != nil && !slices.Equal(got, tt.want) {
				t.Fatalf("distribution = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProductionCommentsStartCollapsed(t *testing.T) {
	t.Parallel()
	templates, err := parseTemplates()
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	rows := []productionEditorRow{
		{ProductName: "Батон"},
		{ProductName: "Багет", Reason: "Подгорело"},
		{ProductName: "Булка", ReasonConflict: true},
	}
	var output bytes.Buffer
	if err := templates.ExecuteTemplate(&output, "production-rows", rows); err != nil {
		t.Fatalf("render production rows: %v", err)
	}
	html := output.String()
	if strings.Count(html, `class="field item-comment" hidden`) != 2 ||
		strings.Count(html, `aria-expanded="false"`) != 2 ||
		strings.Count(html, `aria-expanded="true"`) != 1 {
		t.Fatalf("unexpected comment disclosure states: %s", html)
	}
	for _, heading := range []string{"Название", "Заявка", "Закладка"} {
		if !strings.Contains(html, "<th>"+heading+"</th>") {
			t.Errorf("production table is missing %q column", heading)
		}
	}
	if strings.Contains(html, "production-sources") {
		t.Error("production table exposes per-order allocation details")
	}
}

func assertProductionItem(t *testing.T, order contract.ProductionOrderWrite, number string, loaded, produced float64) {
	t.Helper()
	if order.Number != number || len(order.Items) != 1 {
		t.Fatalf("order = %#v, want %s with one item", order, number)
	}
	item := order.Items[0]
	if item.LoadedQuantity == nil || *item.LoadedQuantity != loaded || item.ProducedQuantity != produced || item.Reason != "Подгорело" {
		t.Fatalf("%s item = %#v", number, item)
	}
}
