package web

import "testing"

func TestFormatQuantityRoundsToTwoDecimals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"dough division tail", 1.3333333333333333, "1.33"},
		{"rounds half up", 2.005, "2.01"},
		{"rounds down", 12.344, "12.34"},
		{"drops trailing zero", 1.5, "1.5"},
		{"rounds to a whole number", 2.999, "3"},
		{"whole number stays integral", 12.0, "12"},
		{"order quantity untouched", 0.1, "0.1"},
		{"below half a hundredth collapses", 0.004, "0"},
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
