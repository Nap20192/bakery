package web

import "testing"

func TestFormatQuantityRoundsToThreeDecimals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"dough division tail", 1.3333333333333333, "1.333"},
		{"rounds half up", 2.0005, "2.001"},
		{"rounds down", 12.3444, "12.344"},
		{"keeps two decimals", 12.34, "12.34"},
		{"drops trailing zero", 1.5, "1.5"},
		{"rounds to a whole number", 2.9999, "3"},
		{"whole number stays integral", 12.0, "12"},
		{"order quantity untouched", 0.1, "0.1"},
		{"below half a thousandth collapses", 0.0004, "0"},
		{"integer input", 7, "7"},
		{"nil pointer", (*float64)(nil), "—"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := formatQuantity(test.value); got != test.want {
				t.Errorf("formatQuantity(%v) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestFormatQuantity3KeepsFixedDecimals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		value float64
		want  string
	}{
		{0.3, "0.300"},
		{0.1, "0.100"},
		{6.7381234, "6.738"},
		{11.1, "11.100"},
		{0, "0.000"},
		{1.3333333333333333, "1.333"},
	}
	for _, test := range tests {
		if got := formatQuantity3(test.value); got != test.want {
			t.Errorf("formatQuantity3(%v) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestActivePathSeparatesNewOrderFromOrders(t *testing.T) {
	t.Parallel()
	if activePath("/orders/new", "/orders") {
		t.Fatal("orders tab is active on the separate new-order page")
	}
	if !activePath("/orders/new", "/orders/new") {
		t.Fatal("new-order tab is not active")
	}
}
