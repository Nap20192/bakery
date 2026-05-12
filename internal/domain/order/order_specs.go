package order

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type BulkOrderLine struct {
	Number int
	Raw    string
}

type ParsedOrderLine struct {
	Line     int
	Raw      string
	Code     string
	Name     string
	Quantity string
}

var (
	orderLineRe         = regexp.MustCompile(`^(\S+)\s+(.+?)\s+(\d+(?:[.,]\d+)?(?:\+\d+(?:[.,]\d+)?)?)$`)
	fulfillmentDateRe   = regexp.MustCompile(`^(?:на|date)\s+(\d{4}-\d{2}-\d{2})$`)
	singleWordLineRe    = regexp.MustCompile(`^\p{L}+$`)
	quantitySeparatorRe = regexp.MustCompile(`\+`)
)

type OrderLineProcessableSpecification struct{}

func (s OrderLineProcessableSpecification) IsValid(line BulkOrderLine) bool {
	return line.Raw != "" && !singleWordLineRe.MatchString(line.Raw)
}

type OrderLineFormatSpecification struct{}

func (s OrderLineFormatSpecification) IsValid(line BulkOrderLine) bool {
	return orderLineRe.MatchString(line.Raw)
}

func ParseOrderLine(line BulkOrderLine) ParsedOrderLine {
	matches := orderLineRe.FindStringSubmatch(line.Raw)
	if len(matches) != 4 {
		return ParsedOrderLine{Line: line.Number, Raw: line.Raw}
	}
	return ParsedOrderLine{
		Line:     line.Number,
		Raw:      line.Raw,
		Code:     matches[1],
		Name:     matches[2],
		Quantity: matches[3],
	}
}

type PositiveQuantitySpecification struct{}

func (s PositiveQuantitySpecification) IsValid(line ParsedOrderLine) bool {
	qty, err := line.QuantityValue()
	if err != nil || qty < 0 {
		return false
	}
	reservedQty, err := line.ReservedQuantityValue()
	if err != nil || reservedQty < 0 {
		return false
	}
	return qty+reservedQty > 0
}

func (line ParsedOrderLine) QuantityValue() (float64, error) {
	parts := quantitySeparatorRe.Split(line.Quantity, -1)
	return parseQuantityPart(parts[0])
}

func (line ParsedOrderLine) ReservedQuantityValue() (float64, error) {
	parts := quantitySeparatorRe.Split(line.Quantity, -1)
	if len(parts) == 1 {
		return 0, nil
	}
	return parseQuantityPart(parts[1])
}

func parseQuantityPart(value string) (float64, error) {
	return strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
}

func ParseFulfillmentDateLine(line string) (time.Time, bool, error) {
	matches := fulfillmentDateRe.FindStringSubmatch(strings.ToLower(strings.TrimSpace(line)))
	if len(matches) != 2 {
		return time.Time{}, false, nil
	}
	date, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return time.Time{}, true, fmt.Errorf("invalid fulfillment date %q", matches[1])
	}
	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if date.Before(today) {
		return time.Time{}, true, fmt.Errorf("fulfillment date %s is in the past", matches[1])
	}
	return date, true, nil
}

type UniqueOrderItemsSpecification struct{}

func (s UniqueOrderItemsSpecification) IsValid(items []OrderItem) bool {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := item.Code
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func DuplicateOrderItemErrors(items []OrderItem) []error {
	seen := make(map[string]struct{}, len(items))
	var errs []error
	for _, item := range items {
		key := item.Code
		if _, exists := seen[key]; exists {
			errs = append(errs, fmt.Errorf("duplicate item with code %s", item.Code))
			continue
		}
		seen[key] = struct{}{}
	}
	return errs
}
