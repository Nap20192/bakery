package bot

import (
	"testing"
	"time"

	orderdomain "bakery/internal/domain/order"
)

func TestOrderListButtonText(t *testing.T) {
	order := orderdomain.Order{
		Number:          "Г.21.05.26.012",
		FulfillmentDate: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
	}

	if got := orderListButtonText(order); got != "Г.21.05.26.012 / 22.05.2026" {
		t.Fatalf("orderListButtonText = %q", got)
	}
}
