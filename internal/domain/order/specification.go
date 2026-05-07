package order

type Specification[T any] interface {
	IsValid(value T) bool
}

type FunctionSpecification[T any] func(value T) bool

func (s FunctionSpecification[T]) IsValid(value T) bool {
	return s(value)
}

type AndSpecification[T any] struct {
	specifications []Specification[T]
}

func NewAndSpecification[T any](specifications ...Specification[T]) Specification[T] {
	return AndSpecification[T]{specifications: specifications}
}

func (s AndSpecification[T]) IsValid(value T) bool {
	for _, specification := range s.specifications {
		if !specification.IsValid(value) {
			return false
		}
	}
	return true
}

type OrSpecification[T any] struct {
	specifications []Specification[T]
}

func NewOrSpecification[T any](specifications ...Specification[T]) Specification[T] {
	return OrSpecification[T]{specifications: specifications}
}

func (s OrSpecification[T]) IsValid(value T) bool {
	for _, specification := range s.specifications {
		if specification.IsValid(value) {
			return true
		}
	}
	return false
}

type NotSpecification[T any] struct {
	specification Specification[T]
}

func NewNotSpecification[T any](specification Specification[T]) Specification[T] {
	return NotSpecification[T]{specification: specification}
}

func (s NotSpecification[T]) IsValid(value T) bool {
	return !s.specification.IsValid(value)
}
