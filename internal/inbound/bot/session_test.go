package bot

import (
	"testing"

	orderdomain "bakery/internal/domain/order"
)

func TestMergeSessionItemsReplacesExistingQuantity(t *testing.T) {
	existing := []orderdomain.OrderItem{
		{Code: "15647", ProductName: "Сосиска в тесте", Quantity: 5},
	}
	incoming := []orderdomain.OrderItem{
		{Code: "15647", ProductName: "Сосиска в тесте", Quantity: 2, ReservedQuantity: 1},
	}

	got := mergeSessionItems(existing, incoming)
	if len(got) != 1 {
		t.Fatalf("items = %d, want 1", len(got))
	}
	if got[0].Quantity != 2 {
		t.Fatalf("quantity = %v, want 2", got[0].Quantity)
	}
	if got[0].ReservedQuantity != 1 {
		t.Fatalf("reserved quantity = %v, want 1", got[0].ReservedQuantity)
	}
}

func TestMergeSessionItemsZeroQuantityRemovesExistingItem(t *testing.T) {
	existing := []orderdomain.OrderItem{
		{Code: "15647", ProductName: "Сосиска в тесте", Quantity: 5},
	}
	incoming := []orderdomain.OrderItem{
		{Code: "15647", ProductName: "Сосиска в тесте", Quantity: 0},
	}

	got := mergeSessionItems(existing, incoming)
	if len(got) != 0 {
		t.Fatalf("items = %d, want 0", len(got))
	}
}
