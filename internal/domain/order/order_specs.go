package order

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	orderLineRe      = regexp.MustCompile(`^(\S+)\s+(.+?)\s+(\d+(?:[.,]\d+)?)$`)
	singleWordLineRe = regexp.MustCompile(`^\p{L}+$`)
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
	qty, err := strconv.ParseFloat(strings.ReplaceAll(line.Quantity, ",", "."), 64)
	return err == nil && qty >= 0
}

func (line ParsedOrderLine) QuantityValue() (float64, error) {
	return strconv.ParseFloat(strings.ReplaceAll(line.Quantity, ",", "."), 64)
}

type UniqueOrderItemsSpecification struct{}

func (s UniqueOrderItemsSpecification) IsValid(items []OrderItem) bool {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := item.Code + "\x00" + item.ProductName
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
		key := item.Code + "\x00" + item.ProductName
		if _, exists := seen[key]; exists {
			errs = append(errs, fmt.Errorf("duplicate item with code %s and product_name %q", item.Code, item.ProductName))
			continue
		}
		seen[key] = struct{}{}
	}
	return errs
}
