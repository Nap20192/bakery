package order

import sharedkernel "bakery/internal/pkg/shared_kernel"

type Specification[T any] = sharedkernel.Specification[T]

type FunctionSpecification[T any] = sharedkernel.FunctionSpecification[T]

type AndSpecification[T any] = sharedkernel.AndSpecification[T]

func NewAndSpecification[T any](specifications ...Specification[T]) Specification[T] {
	return sharedkernel.NewAndSpecification(specifications...)
}

type OrSpecification[T any] = sharedkernel.OrSpecification[T]

func NewOrSpecification[T any](specifications ...Specification[T]) Specification[T] {
	return sharedkernel.NewOrSpecification(specifications...)
}

type NotSpecification[T any] = sharedkernel.NotSpecification[T]

func NewNotSpecification[T any](specification Specification[T]) Specification[T] {
	return sharedkernel.NewNotSpecification(specification)
}
