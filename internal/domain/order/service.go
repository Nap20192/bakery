package order

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type BulkOrderValidationResult struct {
	ValidItems      []OrderItem
	Errors          []BulkOrderValidationError
	FulfillmentDate time.Time
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
	LineProcessable BulkOrderLineSpecification
	LineFormat      BulkOrderLineSpecification
	Quantity        ParsedOrderLineSpecification
	UniqueItems     OrderItemsSpecification
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

func (s *OrderService) NormalizeFulfillmentDate(t time.Time, fallback time.Time) time.Time {
	if t.IsZero() {
		if fallback.IsZero() {
			fallback = time.Now().UTC()
		}
		t = fallback.AddDate(0, 0, 1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
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
		if date, ok, err := ParseFulfillmentDateLine(line); ok {
			if err != nil {
				result.Errors = append(result.Errors, BulkOrderValidationError{
					Line:    i + 1,
					Raw:     line,
					Message: err.Error(),
				})
				continue
			}
			result.FulfillmentDate = date
			continue
		}
		if !s.spec.LineProcessable.IsValid(candidate) {
			continue
		}

		if !s.spec.LineFormat.IsValid(candidate) {
			result.Errors = append(result.Errors, BulkOrderValidationError{
				Line:    i + 1,
				Raw:     line,
				Message: "invalid format: expected code product_name quantity or code product_name quantity+reserved_quantity",
			})
			continue
		}

		parsed := ParseOrderLine(candidate)
		if !s.spec.Quantity.IsValid(parsed) {
			result.Errors = append(result.Errors, BulkOrderValidationError{
				Line:    parsed.Line,
				Code:    parsed.Code,
				Name:    parsed.Name,
				Message: fmt.Sprintf("invalid quantity %q: quantity and reserved_quantity must be non-negative integers", parsed.Quantity),
			})
			continue
		}
		qty, _ := parsed.QuantityValue()
		reservedQty, _ := parsed.ReservedQuantityValue()

		result.ValidItems = append(result.ValidItems, OrderItem{
			Code:             parsed.Code,
			ProductName:      parsed.Name,
			Quantity:         qty,
			ReservedQuantity: reservedQty,
		})
	}
	result.FulfillmentDate = s.NormalizeFulfillmentDate(result.FulfillmentDate, time.Now().UTC())

	return result
}

func (s *OrderService) ValidateUniqueItems(items []OrderItem) error {
	if s.spec.UniqueItems.IsValid(items) {
		return nil
	}
	return errors.Join(DuplicateOrderItemErrors(items)...)
}

func (s *OrderService) IsTemplateHeader(line string) bool {
	return IsTemplateHeaderLine(line)
}

func (s *OrderService) ParseOrderTemplate(raw string) (ParsedOrderTemplate, BulkOrderValidationResult) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var title string
	var bodyLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if title == "" {
			title = line
			continue
		}
		bodyLines = append(bodyLines, line)
	}

	var result BulkOrderValidationResult
	if title == "" || !s.IsTemplateHeader(title) {
		result.Errors = append(result.Errors, BulkOrderValidationError{
			Line:    1,
			Raw:     title,
			Message: "template first line must be an uppercase title",
		})
		return ParsedOrderTemplate{}, result
	}

	result = s.ParseBulkOrder(strings.Join(bodyLines, "\n"))
	if len(result.ValidItems) == 0 {
		result.Errors = append(result.Errors, BulkOrderValidationError{
			Message: "template must contain at least one product line",
		})
	}
	for _, item := range result.ValidItems {
		if item.Quantity != 0 || item.ReservedQuantity != 0 {
			result.Errors = append(result.Errors, BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: "template quantity must be 0",
			})
		}
	}
	if len(result.Errors) > 0 {
		return ParsedOrderTemplate{}, result
	}
	body := title
	if len(bodyLines) > 0 {
		body += "\n" + strings.Join(bodyLines, "\n")
	}
	return ParsedOrderTemplate{
		Name:  title,
		Body:  body,
		Items: result.ValidItems,
	}, result
}
