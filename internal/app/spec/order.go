package spec

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"bakery/internal/domain"
	"bakery/internal/repo/sqlc"
)if err != nil {
		return false, err
	}

type OrderSpec struct {
	queries sqlc.Querier
}

func NewOrderSpec(queries sqlc.Querier) *OrderSpec {
	return &OrderSpec{queries: queries}
}

type BulkOrderValidationResult struct {
	Items      []BulkOrderValidationItem
	ValidItems []domain.OrderItem
	Errors     []string
}

func (r BulkOrderValidationResult) Valid() bool {
	return len(r.Errors) == 0
}

type BulkOrderValidationItem struct {
	LineNumber    int
	Raw           string
	ProductCode   string
	ProductName   string
	Quantity      float64
	IikoProductID *string
	CanonicalName string
	Errors        []string
}

func (s *OrderSpec) ValidateBulkOrder(ctx context.Context, batchOrder []byte) (BulkOrderValidationResult, error) {
	lines := parseBulkOrderSpec(string(batchOrder))
	result := BulkOrderValidationResult{
		Items: make([]BulkOrderValidationItem, 0, len(lines)),
	}
	if len(lines) == 0 {
		result.Errors = append(result.Errors, "order is empty")
		return result, nil
	}

	duplicatesInOrder := duplicateCodes(lines)

	for _, line := range lines {
		item := BulkOrderValidationItem{
			LineNumber:  line.LineNumber,
			Raw:         line.Raw,
			ProductCode: line.ProductCode,
			ProductName: line.ProductName,
			Quantity:    line.Quantity,
		}

		product, found, err := s.getProductByCode(ctx, line.ProductCode)
		if err != nil {
			return BulkOrderValidationResult{}, err
		}
		if found {
			item.IikoProductID = &product.ID
			item.CanonicalName = product.Name
		}

		item.Errors = append(item.Errors, s.validateLine(line, duplicatesInOrder, product, found)...)

		if len(item.Errors) == 0 {
			result.ValidItems = append(result.ValidItems, domain.OrderItem{
				Product:       product.Name,
				Code:          product.Code,
				Quantity:      line.Quantity,
				IikoProductID: &product.ID,
			})
		} else {
			for _, msg := range item.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("line %d: %s", item.LineNumber, msg))
			}
		}

		result.Items = append(result.Items, item)
	}

	return result, nil
}

func (s *OrderSpec) getProductByCode(ctx context.Context, code string) (sqlc.GetIikoProductByCodeRow, bool, error) {
	if code == "" {
		return sqlc.GetIikoProductByCodeRow{}, false, nil
	}
	product, err := s.queries.GetIikoProductByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.GetIikoProductByCodeRow{}, false, nil
		}
		return sqlc.GetIikoProductByCodeRow{}, false, fmt.Errorf("get iiko product by code %q: %w", code, err)
	}
	return product, true, nil
}

func (s *OrderSpec) validateLine(
	line parsedBulkOrderSpecLine,
	duplicatesInOrder map[string]struct{},
	product sqlc.GetIikoProductByCodeRow,
	productExists bool,
) []string {
	if line.ParseError != "" {
		return []string{line.ParseError}
	}

	var errors []string
	if _, duplicate := duplicatesInOrder[line.ProductCode]; duplicate {
		errors = append(errors, "duplicate product code in order")
	}

	if !productExists {
		errors = append(errors, "product code not found")
		return errors
	}
	if !sameProductName(product.Name, line.ProductName) {
		errors = append(errors, fmt.Sprintf("product name mismatch: expected %q", product.Name))
	}
	return errors
}

type parsedBulkOrderSpecLine struct {
	LineNumber  int
	Raw         string
	ProductCode string
	ProductName string
	Quantity    float64
	ParseError  string
}

func parseBulkOrderSpec(text string) []parsedBulkOrderSpecLine {
	var result []parsedBulkOrderSpecLine
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		parsed := parsedBulkOrderSpecLine{
			LineNumber: i + 1,
			Raw:        raw,
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			parsed.ParseError = "expected format: product_code product_name quantity"
			result = append(result, parsed)
			continue
		}

		quantity, err := strconv.ParseFloat(strings.ReplaceAll(fields[len(fields)-1], ",", "."), 64)
		if err != nil || quantity <= 0 {
			parsed.ParseError = "quantity must be a positive number"
			result = append(result, parsed)
			continue
		}

		parsed.ProductCode = fields[0]
		parsed.ProductName = strings.Join(fields[1:len(fields)-1], " ")
		parsed.Quantity = quantity
		result = append(result, parsed)
	}
	return result
}

func duplicateCodes(lines []parsedBulkOrderSpecLine) map[string]struct{} {
	counts := make(map[string]int, len(lines))
	for _, line := range lines {
		if line.ProductCode != "" {
			counts[line.ProductCode]++
		}
	}

	duplicates := make(map[string]struct{})
	for code, count := range counts {
		if count > 1 {
			duplicates[code] = struct{}{}
		}
	}
	return duplicates
}

func sameProductName(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}
