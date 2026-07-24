package web

import (
	"bytes"
	"net/http/httptest"
	"net/url"
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

	rounded := distributeProductionQuantity(1, []float64{1, 1, 1})
	if rounded[0] != 0.3 || rounded[1] != 0.3 || rounded[2] != 0.4 {
		t.Fatalf("rounded distribution = %v, want [0.3 0.3 0.4]", rounded)
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
