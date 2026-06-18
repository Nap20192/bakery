package bot

import (
	"strings"
	"testing"
	"time"

	orderdomain "bakery/internal/services/order/domain"
)

func TestOrderViewShowsOrderItemsAsCopyableCodeBlock(t *testing.T) {
	order := orderdomain.Order{
		Number: "Г.24.05.26.001",
		Items: []orderdomain.OrderItem{
			{Code: "15647", ProductName: "Сосиска в тесте", Quantity: 5, ReservedQuantity: 2},
		},
	}

	got := responses.OrderView(order, "Магазин Гагарина", "Цех Пекари")

	want := "<b>Состав заказа:</b>\n<pre>15647 Сосиска в тесте 5+2\n</pre>"
	if !strings.Contains(got, want) {
		t.Fatalf("OrderView() = %q, want to contain %q", got, want)
	}
}

func TestOrderDraftShowsOrderItemsAsCopyableCodeBlock(t *testing.T) {
	items := []orderdomain.OrderItem{
		{Code: "15541", ProductName: "Кокрок с капустой", Quantity: 3},
	}

	got := responses.OrderDraft("", items, time.Time{}, nil)

	want := "<pre>15541 Кокрок с капустой 3\n</pre>"
	if !strings.Contains(got, want) {
		t.Fatalf("OrderDraft() = %q, want to contain %q", got, want)
	}
}
