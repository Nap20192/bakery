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

// BuildOrderNumber composes the human-facing order number:
// <буква отправителя>.<буква категории>.<дата>.<счётчик>, e.g. «Г.Х.08.07.26.001».
// An empty category letter keeps the legacy format without the extra segment.
func (s *OrderService) BuildOrderNumber(sourceCode string, sourceName string, categoryLetter string, createdAt time.Time, counter int64) string {
	prefix := orderSourcePrefix(sourceCode, sourceName)
	if letter := strings.TrimSpace(categoryLetter); letter != "" {
		prefix += "." + letter
	}
	return fmt.Sprintf("%s.%s.%03d", prefix, s.NormalizeCreatedAt(createdAt).Format("02.01.06"), counter)
}

func orderSourcePrefix(sourceCode string, sourceName string) string {
	switch strings.ToLower(strings.TrimSpace(sourceCode)) {
	case "gagarina":
		return "Г"
	case "sholokhova":
		return "Ш"
	case "saryarka":
		return "С"
	}

	normalizedName := strings.ToLower(strings.TrimSpace(sourceName))
	switch {
	case strings.Contains(normalizedName, "гагарина"):
		return "Г"
	case strings.Contains(normalizedName, "шолохова"):
		return "Ш"
	case strings.Contains(normalizedName, "сарыарка"):
		return "С"
	}

	for _, r := range strings.TrimSpace(sourceName) {
		if unicode.IsSpace(r) {
			continue
		}
		return string(unicode.ToUpper(r))
	}
	return ""
}

// splitLineComment separates an optional per-item comment from a bulk line.
// Comment starts at the first "//" or ";" separator, e.g.
// "Кокрок 5 // посыпать сахаром".
func splitLineComment(line string) (string, string) {
	best := -1
	bestLen := 0
	for _, sep := range []string{"//", ";"} {
		if i := strings.Index(line, sep); i >= 0 && (best < 0 || i < best) {
			best = i
			bestLen = len(sep)
		}
	}
	if best < 0 {
		return line, ""
	}
	return strings.TrimSpace(line[:best]), strings.TrimSpace(line[best+bestLen:])
}

func (s *OrderService) ParseBulkOrder(order string) BulkOrderValidationResult {
	var result BulkOrderValidationResult
	lines := strings.Split(order, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		line, comment := splitLineComment(line)
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
				Message: "Не распознано. Формат: название количество (напр. Сосиска в тесте 5+2).",
			})
			continue
		}

		parsed := ParseOrderLine(candidate)
		if !s.spec.Quantity.IsValid(parsed) {
			result.Errors = append(result.Errors, BulkOrderValidationError{
				Line:    parsed.Line,
				Code:    parsed.Code,
				Name:    parsed.Name,
				Message: fmt.Sprintf("Количество %q не подходит. Нужно целое (напр. 5 или 5+2).", parsed.Quantity),
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
			Comment:          comment,
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
