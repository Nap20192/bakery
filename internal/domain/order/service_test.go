package order

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOrderServiceParseBulkOrder(t *testing.T) {
	svc := NewOrderService()

	result := svc.ParseBulkOrder(`
ПИРОЖКИ
15635 Пирожок с капустой 15
20495 Пирожок с картошкой 2,5
broken line
15647 Сосиска в тесте abc
15648 Сосиска с сыром в тесте 0
`)

	if len(result.ValidItems) != 3 {
		t.Fatalf("valid items = %d, want 3: %#v", len(result.ValidItems), result.ValidItems)
	}

	assertItem(t, result.ValidItems[0], "15635", "Пирожок с капустой", 15)
	assertItem(t, result.ValidItems[1], "20495", "Пирожок с картошкой", 2.5)
	assertItem(t, result.ValidItems[2], "15648", "Сосиска с сыром в тесте", 0)

	if len(result.Errors) != 2 {
		t.Fatalf("errors = %d, want 2: %#v", len(result.Errors), result.Errors)
	}
	if result.Errors[0].Line != 5 || result.Errors[0].Raw != "broken line" || result.Errors[0].Message != "invalid format" {
		t.Fatalf("unexpected first error: %#v", result.Errors[0])
	}
	if result.Errors[1].Line != 6 || result.Errors[1].Raw != "15647 Сосиска в тесте abc" || result.Errors[1].Message != "invalid format" {
		t.Fatalf("unexpected second error: %#v", result.Errors[1])
	}
}

func TestOrderServiceUsesInjectedSpec(t *testing.T) {
	svc := &OrderService{spec: OrderSpec{
		LineProcessable: FunctionSpecification[BulkOrderLine](func(BulkOrderLine) bool { return true }),
		LineFormat:      FunctionSpecification[BulkOrderLine](func(BulkOrderLine) bool { return false }),
		Quantity:        PositiveQuantitySpecification{},
		UniqueItems:     UniqueOrderItemsSpecification{},
	}}

	result := svc.ParseBulkOrder("15635 Пирожок с капустой 1")

	if len(result.ValidItems) != 0 {
		t.Fatalf("valid items = %d, want 0", len(result.ValidItems))
	}
	if len(result.Errors) != 1 || result.Errors[0].Message != "invalid format" {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
}

func TestOrderServiceValidateUniqueItems(t *testing.T) {
	svc := NewOrderService()

	err := svc.ValidateUniqueItems([]OrderItem{
		{Code: "15635", ProductName: "Пирожок с капустой", Quantity: 1},
		{Code: "15635", ProductName: "Пирожок с капустой", Quantity: 2},
		{Code: "15635", ProductName: "Другое имя", Quantity: 1},
	})

	if err == nil {
		t.Fatal("expected duplicate error")
	}
	if !strings.Contains(err.Error(), `duplicate item with code 15635 and product_name "Пирожок с капустой"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.ValidateUniqueItems([]OrderItem{{Code: "1", ProductName: "A"}, {Code: "1", ProductName: "B"}}); err != nil {
		t.Fatalf("unique validation returned error: %v", err)
	}
}

func TestOrderServiceOrderNumberRules(t *testing.T) {
	svc := NewOrderService()
	createdAt := time.Date(2026, 5, 8, 10, 30, 0, 0, time.FixedZone("ALMT", 5*60*60))

	if got := svc.OrderCounterDay(createdAt); got != "08052026" {
		t.Fatalf("OrderCounterDay = %s, want 08052026", got)
	}
	if got := svc.BuildOrderNumber("08052026", 12); got != "08052026_ORDER_0012" {
		t.Fatalf("BuildOrderNumber = %s", got)
	}
	if got := svc.NormalizeCreatedAt(createdAt); got.Location() != time.UTC {
		t.Fatalf("NormalizeCreatedAt location = %v, want UTC", got.Location())
	}
	if got := svc.NormalizeCreatedAt(time.Time{}); got.IsZero() {
		t.Fatal("NormalizeCreatedAt zero time should use current UTC time")
	}
}

func TestOrderServiceIsTemplateHeader(t *testing.T) {
	svc := NewOrderService()

	tests := []struct {
		line string
		want bool
	}{
		{line: "ПИРОЖКИ", want: true},
		{line: "ПИРОГИ СЫТНЫЕ/СЛАДКИЕ", want: true},
		{line: "Кокрок с картофелем", want: false},
		{line: "15635 Пирожок с капустой 15", want: false},
		{line: "12345", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := svc.IsTemplateHeader(tt.line); got != tt.want {
				t.Fatalf("IsTemplateHeader(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestPositiveQuantitySpecification(t *testing.T) {
	spec := PositiveQuantitySpecification{}

	if !spec.IsValid(ParsedOrderLine{Quantity: "1,25"}) {
		t.Fatal("comma decimal should be valid")
	}
	if !spec.IsValid(ParsedOrderLine{Quantity: "0"}) {
		t.Fatal("zero quantity should be valid")
	}
	if spec.IsValid(ParsedOrderLine{Quantity: "-1"}) {
		t.Fatal("negative quantity should be invalid")
	}
	if spec.IsValid(ParsedOrderLine{Quantity: "abc"}) {
		t.Fatal("non numeric quantity should be invalid")
	}
}

func TestDuplicateOrderItemErrors(t *testing.T) {
	errs := DuplicateOrderItemErrors([]OrderItem{
		{Code: "1", ProductName: "A"},
		{Code: "2", ProductName: "A"},
		{Code: "1", ProductName: "A"},
		{Code: "1", ProductName: "A"},
	})
	if len(errs) != 2 {
		t.Fatalf("duplicate errors = %d, want 2", len(errs))
	}
	if !errors.Is(errors.Join(errs...), errs[0]) {
		t.Fatal("joined duplicate errors should preserve children")
	}
}

func assertItem(t *testing.T, item OrderItem, code string, name string, quantity float64) {
	t.Helper()
	if item.Code != code || item.ProductName != name || item.Quantity != quantity {
		t.Fatalf("item = %#v, want code=%s name=%s quantity=%v", item, code, name, quantity)
	}
}
