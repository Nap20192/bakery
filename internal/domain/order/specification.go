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

type ParsedOrderLineSpecificationFunc func(line ParsedOrderLine) bool

func (s ParsedOrderLineSpecificationFunc) IsValid(line ParsedOrderLine) bool {
	return s(line)
}

type OrderItemsSpecificationFunc func(items []OrderItem) bool

func (s OrderItemsSpecificationFunc) IsValid(items []OrderItem) bool {
	return s(items)
}
