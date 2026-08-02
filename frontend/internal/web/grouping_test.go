package web

import (
	"testing"

	"bakery/internal/inbound/api/contract"
)

// Proves theme grouping is really applied in both editors: dishes split by
// theme, one group per distinct theme, empty theme collapses into «Без группы»,
// and a group exists only because it has at least one dish.
func TestEditorGroupsByTheme(t *testing.T) {
	catalog := []contract.Dish{
		{Code: "a", Name: "Багет", Theme: "Хлеб"},
		{Code: "b", Name: "Чиабатта", Theme: "Хлеб"},
		{Code: "c", Name: "Синнабон", Theme: "Булочки"},
		{Code: "d", Name: "Ноунейм", Theme: ""},
	}

	groups := buildEditorGroups(catalog, contract.OrderWrite{})
	if len(groups) != 3 {
		t.Fatalf("want 3 groups, got %d", len(groups))
	}
	want := []struct {
		name string
		size int
	}{{"Хлеб", 2}, {"Булочки", 1}, {"Без группы", 1}}
	for i, w := range want {
		if groups[i].Name != w.name || len(groups[i].Items) != w.size {
			t.Errorf("group %d = %q(%d), want %q(%d)", i, groups[i].Name, len(groups[i].Items), w.name, w.size)
		}
	}

	// Same grouping backs the admin catalog panels and the calculator path.
	themes := groupDishesByTheme(catalog)
	if len(themes) != 3 || themes[0].Name != "Хлеб" || len(themes[0].Dishes) != 2 {
		t.Fatalf("groupDishesByTheme mismatch: %+v", themes)
	}
}
