package order

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

type BulkOrderValidationResult struct {
	ValidItems []OrderItem
	Errors     []BulkOrderValidationError
}

type BulkOrderValidationError struct {
	Line    int
	Raw     string
	Code    string
	Name    string
	Message string
}

type OrderService struct {
	spec OrderSpec
}

type OrderSpec struct {
	LineProcessable Specification[BulkOrderLine]
	LineFormat      Specification[BulkOrderLine]
	Quantity        Specification[ParsedOrderLine]
	UniqueItems     Specification[[]OrderItem]
}

func NewOrderService() *OrderService {
	return &OrderService{spec: NewOrderSpec()}
}

func NewOrderSpec() OrderSpec {
	return OrderSpec{
		LineProcessable: OrderLineProcessableSpecification{},
		LineFormat:      OrderLineFormatSpecification{},
		Quantity:        PositiveQuantitySpecification{},
		UniqueItems:     UniqueOrderItemsSpecification{},
	}
}

func (s *OrderService) NormalizeCreatedAt(t time.Time) time.Time {
	createdAt := t.UTC()
	if createdAt.IsZero() {
		return time.Now().UTC()
	}
	return createdAt
}

func (s *OrderService) OrderCounterDay(t time.Time) string {
	return s.NormalizeCreatedAt(t).Format("02012006")
}

func (s *OrderService) BuildOrderNumber(day string, counter int64) string {
	return fmt.Sprintf("%s_ORDER_%04d", day, counter)
}

func (s *OrderService) ParseBulkOrder(order string) BulkOrderValidationResult {
	var result BulkOrderValidationResult
	lines := strings.Split(order, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		candidate := BulkOrderLine{Number: i + 1, Raw: line}
		if !s.spec.LineProcessable.IsValid(candidate) {
			continue
		}

		if !s.spec.LineFormat.IsValid(candidate) {
			result.Errors = append(result.Errors, BulkOrderValidationError{
				Line:    i + 1,
				Raw:     line,
				Message: "invalid format",
			})
			continue
		}

		parsed := ParseOrderLine(candidate)
		if !s.spec.Quantity.IsValid(parsed) {
			result.Errors = append(result.Errors, BulkOrderValidationError{
				Line:    parsed.Line,
				Code:    parsed.Code,
				Name:    parsed.Name,
				Message: fmt.Sprintf("invalid quantity %q", parsed.Quantity),
			})
			continue
		}
		qty, _ := parsed.QuantityValue()

		result.ValidItems = append(result.ValidItems, OrderItem{
			Code:        parsed.Code,
			ProductName: parsed.Name,
			Quantity:    qty,
		})
	}

	return result
}

func (s *OrderService) ValidateUniqueItems(items []OrderItem) error {
	if s.spec.UniqueItems.IsValid(items) {
		return nil
	}
	return errors.Join(DuplicateOrderItemErrors(items)...)
}

func (s *OrderService) IsTemplateHeader(line string) bool {
	hasLetter := false
	for _, r := range line {
		if !unicode.IsLetter(r) {
			continue
		}
		hasLetter = true
		if unicode.IsLower(r) {
			return false
		}
	}
	return hasLetter
}
