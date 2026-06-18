package order

type BulkOrderLineSpecification interface {
	IsValid(line BulkOrderLine) bool
}

type ParsedOrderLineSpecification interface {
	IsValid(line ParsedOrderLine) bool
}

type OrderItemsSpecification interface {
	IsValid(items []OrderItem) bool
}

type BulkOrderLineSpecificationFunc func(line BulkOrderLine) bool

func (s BulkOrderLineSpecificationFunc) IsValid(line BulkOrderLine) bool {
	return s(line)
}
