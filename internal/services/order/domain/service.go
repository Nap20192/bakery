package order

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
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

func (s *OrderService) BuildOrderNumber(shopCode string, shopName string, createdAt time.Time, counter int64) string {
	return fmt.Sprintf("%s.%s.%03d", shopOrderPrefix(shopCode, shopName), s.NormalizeCreatedAt(createdAt).Format("02.01.06"), counter)
}

func shopOrderPrefix(shopCode string, shopName string) string {
	switch strings.ToLower(strings.TrimSpace(shopCode)) {
	case "gagarina":
		return "Г"
	case "sholokhova":
		return "Ш"
	case "saryarka":
		return "С"
	}

	normalizedName := strings.ToLower(strings.TrimSpace(shopName))
	switch {
	case strings.Contains(normalizedName, "гагарина"):
		return "Г"
	case strings.Contains(normalizedName, "шолохова"):
		return "Ш"
	case strings.Contains(normalizedName, "сарыарка"):
		return "С"
	}

	for _, r := range strings.TrimSpace(shopName) {
		if unicode.IsSpace(r) {
			continue
		}
		return string(unicode.ToUpper(r))
	}
	return ""
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
				Message: "Строка не распознана. Формат: название количество. Например: Сосиска в тесте 5. Заказное количество пишите через плюс: 5+2.",
			})
			continue
		}

		parsed := ParseOrderLine(candidate)
		if !s.spec.Quantity.IsValid(parsed) {
			result.Errors = append(result.Errors, BulkOrderValidationError{
				Line:    parsed.Line,
				Code:    parsed.Code,
				Name:    parsed.Name,
				Message: fmt.Sprintf("Количество %q не подходит. Укажите целое число без дробей, например 5 или 5+2.", parsed.Quantity),
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
