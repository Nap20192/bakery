package order

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestOrderServiceParseBulkOrder(t *testing.T) {
	svc := NewOrderService()
	fulfillmentDate := time.Now().UTC().AddDate(0, 0, 1)
	fulfillmentDateText := fulfillmentDate.Format("2006-01-02")
	fulfillmentDateInput := fulfillmentDate.Format("02.01.2006")

	result := svc.ParseBulkOrder(fmt.Sprintf(`
%s
ПИРОЖКИ
ПИРОГИ СЫТНЫЕ/СЛАДКИЕ
15635 Пирожок с капустой 15
Сосиска в тесте 4
20495 Пирожок с картошкой 2,5+1,5
broken line
15647 Сосиска в тесте abc
СДОБНЫЕ ИЗДЕЛИЯ
15648 Сосиска с сыром в тесте 0+3
`, fulfillmentDateInput))

	if len(result.ValidItems) != 3 {
		t.Fatalf("valid items = %d, want 3: %#v", len(result.ValidItems), result.ValidItems)
	}
	if got := result.FulfillmentDate.Format("2006-01-02"); got != fulfillmentDateText {
		t.Fatalf("fulfillment date = %s, want %s", got, fulfillmentDateText)
	}

	assertItem(t, result.ValidItems[0], "15635", "Пирожок с капустой", 15, 0)
	assertItem(t, result.ValidItems[1], "", "Сосиска в тесте", 4, 0)
	assertItem(t, result.ValidItems[2], "15648", "Сосиска с сыром в тесте", 0, 3)

	if len(result.Errors) != 3 {
		t.Fatalf("errors = %d, want 3: %#v", len(result.Errors), result.Errors)
	}
	if result.Errors[0].Line != 7 || result.Errors[0].Code != "20495" || !strings.Contains(result.Errors[0].Message, "целое") {
		t.Fatalf("unexpected first error: %#v", result.Errors[0])
	}
	if result.Errors[1].Line != 8 || result.Errors[1].Raw != "broken line" || !strings.Contains(result.Errors[1].Message, "Не распознано") {
		t.Fatalf("unexpected second error: %#v", result.Errors[1])
	}
	if result.Errors[2].Line != 9 || result.Errors[2].Raw != "15647 Сосиска в тесте abc" || !strings.Contains(result.Errors[2].Message, "Не распознано") {
		t.Fatalf("unexpected third error: %#v", result.Errors[2])
	}
}

func TestOrderServiceUsesInjectedSpec(t *testing.T) {
	svc := &OrderService{spec: OrderSpec{
		LineProcessable: BulkOrderLineSpecificationFunc(func(BulkOrderLine) bool { return true }),
		LineFormat:      BulkOrderLineSpecificationFunc(func(BulkOrderLine) bool { return false }),
		Quantity:        PositiveQuantitySpecification{},
		UniqueItems:     UniqueOrderItemsSpecification{},
	}}

	result := svc.ParseBulkOrder("15635 Пирожок с капустой 1")

	if len(result.ValidItems) != 0 {
		t.Fatalf("valid items = %d, want 0", len(result.ValidItems))
	}
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0].Message, "Не распознано") {
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
	if !strings.Contains(err.Error(), `Позиция с кодом 15635 повторяется`) {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.ValidateUniqueItems([]OrderItem{{Code: "1", ProductName: "A"}, {Code: "2", ProductName: "A"}}); err != nil {
		t.Fatalf("unique validation returned error: %v", err)
	}
}

func TestOrderServiceOrderNumberRules(t *testing.T) {
	svc := NewOrderService()
	createdAt := time.Date(2026, 5, 8, 10, 30, 0, 0, time.FixedZone("ALMT", 5*60*60))

	if got := svc.OrderCounterDay(createdAt); got != "08052026" {
		t.Fatalf("OrderCounterDay = %s, want 08052026", got)
	}
	tests := []struct {
		code string
		name string
		want string
	}{
		{code: "gagarina", name: "Магазин Гагарина", want: "Г.08.05.26.012"},
		{code: "sholokhova", name: "Магазин Шолохова", want: "Ш.08.05.26.012"},
		{code: "saryarka", name: "Магазин Сарыарка", want: "С.08.05.26.012"},
		{code: "", name: "Магазин Гагарина", want: "Г.08.05.26.012"},
	}
	for _, tt := range tests {
		if got := svc.BuildOrderNumber(tt.code, tt.name, createdAt, 12); got != tt.want {
			t.Fatalf("BuildOrderNumber(%q, %q) = %s, want %s", tt.code, tt.name, got, tt.want)
		}
	}
	if got := svc.NormalizeCreatedAt(createdAt); got.Location() != time.UTC {
		t.Fatalf("NormalizeCreatedAt location = %v, want UTC", got.Location())
	}
	if got := svc.NormalizeCreatedAt(time.Time{}); got.IsZero() {
		t.Fatal("NormalizeCreatedAt zero time should use current UTC time")
	}
}

func TestOrderLineProcessableSpecificationSkipsUppercaseHeaders(t *testing.T) {
	spec := OrderLineProcessableSpecification{}

	headers := []string{
		"ПИРОГИ СЫТНЫЕ/СЛАДКИЕ",
		"РЖАНЫЕ ПИРОЖКИ",
		"СДОБНЫЕ ИЗДЕЛИЯ",
	}
	for _, header := range headers {
		if spec.IsValid(BulkOrderLine{Raw: header}) {
			t.Fatalf("header %q should not be processable", header)
		}
	}

	if !spec.IsValid(BulkOrderLine{Raw: "broken line"}) {
		t.Fatal("mixed lowercase invalid order line should stay processable and produce format error")
	}
}

func TestPositiveQuantitySpecification(t *testing.T) {
	spec := PositiveQuantitySpecification{}

	if spec.IsValid(ParsedOrderLine{Quantity: "1,25"}) {
		t.Fatal("comma decimal should be invalid")
	}
	if spec.IsValid(ParsedOrderLine{Quantity: "1.25"}) {
		t.Fatal("dot decimal should be invalid")
	}
	if !spec.IsValid(ParsedOrderLine{Quantity: "0+5"}) {
		t.Fatal("reserved-only quantity should be valid")
	}
	if spec.IsValid(ParsedOrderLine{Quantity: "5+1,25"}) {
		t.Fatal("decimal reserved quantity should be invalid")
	}
	if !spec.IsValid(ParsedOrderLine{Quantity: "5+0"}) {
		t.Fatal("zero reserved quantity should be valid")
	}
	if !spec.IsValid(ParsedOrderLine{Quantity: "0"}) {
		t.Fatal("zero total quantity should be valid as delete marker")
	}
	if !spec.IsValid(ParsedOrderLine{Quantity: "0+0"}) {
		t.Fatal("zero total quantity with reserved should be valid as delete marker")
	}
	if spec.IsValid(ParsedOrderLine{Quantity: "-1"}) {
		t.Fatal("negative quantity should be invalid")
	}
	if spec.IsValid(ParsedOrderLine{Quantity: "abc"}) {
		t.Fatal("non numeric quantity should be invalid")
	}
}

func TestOrderLineFormatSpecificationReservedQuantity(t *testing.T) {
	spec := OrderLineFormatSpecification{}

	valid := []string{
		"15647 Сосиска в тесте 5",
		"15647 Сосиска в тесте 5+5",
		"15647 Сосиска в тесте 0+5",
		"Сосиска в тесте 5",
	}
	for _, line := range valid {
		if !spec.IsValid(BulkOrderLine{Raw: line}) {
			t.Fatalf("line %q should be valid", line)
		}
	}

	invalid := []string{
		"15647 Сосиска в тесте +5",
		"15647 Сосиска в тесте 5+",
		"15647 Сосиска в тесте 5+abc",
		"15647 Сосиска в тесте 5+5+5",
	}
	for _, line := range invalid {
		if spec.IsValid(BulkOrderLine{Raw: line}) {
			t.Fatalf("line %q should be invalid", line)
		}
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

func assertItem(t *testing.T, item OrderItem, code string, name string, quantity float64, reservedQuantity float64) {
	t.Helper()
	if item.Code != code || item.ProductName != name || item.Quantity != quantity || item.ReservedQuantity != reservedQuantity {
		t.Fatalf("item = %#v, want code=%s name=%s quantity=%v reserved_quantity=%v", item, code, name, quantity, reservedQuantity)
	}
}
