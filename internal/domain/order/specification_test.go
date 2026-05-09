package order

import "testing"

func TestSpecificationCombinators(t *testing.T) {
	isEven := FunctionSpecification[int](func(value int) bool { return value%2 == 0 })
	isPositive := FunctionSpecification[int](func(value int) bool { return value > 0 })

	tests := []struct {
		name string
		spec Specification[int]
		in   int
		want bool
	}{
		{name: "and true", spec: NewAndSpecification[int](isEven, isPositive), in: 4, want: true},
		{name: "and false", spec: NewAndSpecification[int](isEven, isPositive), in: -2, want: false},
		{name: "or true", spec: NewOrSpecification[int](isEven, isPositive), in: 3, want: true},
		{name: "or false", spec: NewOrSpecification[int](isEven, isPositive), in: -3, want: false},
		{name: "not", spec: NewNotSpecification[int](isEven), in: 3, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.IsValid(tt.in); got != tt.want {
				t.Fatalf("IsValid(%d) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
