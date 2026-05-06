package app

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"bakery/internal/app/spec"
	"bakery/internal/domain"
	"bakery/internal/repo/sqlc"
)

type OrderService struct {
	queries   *sqlc.Queries
	orderSpec *spec.OrderSpec
}

func NewOrderService(queries *sqlc.Queries) *OrderService {
	return &OrderService{
		queries:   queries,
		orderSpec: spec.NewOrderSpec(queries),
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input domain.CreateOrderInput) (domain.Order, error) {
	if len(input.Items) == 0 {
		return domain.Order{}, fmt.Errorf("order must contain items")
	}

	createdAt := input.Date.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	day := createdAt.Format("02012006")

	if err := s.queries.CreateOrderCounterDay(ctx, day); err != nil {
		return domain.Order{}, fmt.Errorf("init order counter: %w", err)
	}
	counter, err := s.queries.NextOrderCounter(ctx, day)
	if err != nil {
		return domain.Order{}, fmt.Errorf("increment order counter: %w", err)
	}

	number := fmt.Sprintf("%s_ORDER_%04d", day, counter)
	row, err := s.queries.CreateOrder(ctx, sqlc.CreateOrderParams{
		Number:    number,
		Location:  input.Location,
		CreatedAt: createdAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return domain.Order{}, fmt.Errorf("create order: %w", err)
	}

	for _, item := range input.Items {
		var productID *string
		if item.Code != "" {
			product, err := s.queries.GetIikoProductByCode(ctx, item.Code)
			if err != nil && err != sql.ErrNoRows {
				return domain.Order{}, fmt.Errorf("resolve product by code %s: %w", item.Code, err)
			}
			if err == nil {
				productID = &product.ID
			}
		}
		if _, err := s.queries.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
			OrderID:       row.ID,
			IikoProductID: productID,
			ProductName:   item.ProductName,
			Quantity:      item.Quantity,
		}); err != nil {
			return domain.Order{}, fmt.Errorf("create order item: %w", err)
		}
	}

	return domain.Order{
		ID:        fmt.Sprintf("%d", row.ID),
		Number:    row.Number,
		Location:  row.Location,
		Items:     input.Items,
		CreatedAt: parseRFC3339(row.CreatedAt),
	}, nil
}

func (s *OrderService) GetOrderByNumber(ctx context.Context, number string) (domain.Order, error) {
	order, err := s.queries.GetOrderByNumber(ctx, number)
	if err != nil {
		return domain.Order{}, err
	}
	items, err := s.queries.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return domain.Order{}, err
	}
	return domain.Order{
		ID:        fmt.Sprintf("%d", order.ID),
		Number:    order.Number,
		Location:  order.Location,
		Items:     mapOrderItems(items),
		CreatedAt: parseRFC3339(order.CreatedAt),
	}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, limit int32) ([]domain.Order, error) {
	rows, err := s.queries.ListOrders(ctx, int64(limit))
	if err != nil {
		return nil, err
	}
	result := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		items, err := s.queries.GetOrderItemsByOrderID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.Order{
			ID:        fmt.Sprintf("%d", row.ID),
			Number:    row.Number,
			Location:  row.Location,
			Items:     mapOrderItems(items),
			CreatedAt: parseRFC3339(row.CreatedAt),
		})
	}
	return result, nil
}

func (s *OrderService) ResolveDishByName(ctx context.Context, name string) (domain.OrderItem, error) {
	candidates, err := s.queries.GetIikoProductsByName(ctx, strings.TrimSpace(name))
	if err != nil {
		return domain.OrderItem{}, err
	}
	var matched []sqlc.GetIikoProductsByNameRow
	for _, item := range candidates {
		if item.Type != nil && strings.EqualFold(*item.Type, "dish") {
			matched = append(matched, item)
		}
	}
	if len(matched) == 0 {
		return domain.OrderItem{}, fmt.Errorf("dish %q not found", name)
	}
	if len(matched) > 1 {
		sort.Slice(matched, func(i, j int) bool { return matched[i].Code < matched[j].Code })
		return domain.OrderItem{}, fmt.Errorf("dish %q is ambiguous", name)
	}
	return domain.OrderItem{
		Code:        matched[0].Code,
		ProductName: matched[0].Name,
	}, nil
}

type BulkOrderValidationResult struct {
	ValidItems []domain.OrderItem
	Errors     []string
}

var lineRe = regexp.MustCompile(`^(\S+)\s+(.+?)\s+(\d+(?:[.,]\d+)?)$`)
var singleWordRe = regexp.MustCompile(`^\p{L}+$`)

func (s *OrderService) ValidateBulkOrder(ctx context.Context, order string) BulkOrderValidationResult {
	var result BulkOrderValidationResult

	lines := strings.Split(order, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}
		if singleWordRe.MatchString(line) {
			continue
		}

		matches := lineRe.FindStringSubmatch(line)
		if matches == nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("line %d: invalid format", i+1))
			continue
		}

		number := matches[1]
		name := matches[2]
		qtyStr := matches[3]

		qty, err := strconv.ParseFloat(strings.ReplaceAll(qtyStr, ",", "."), 64)

		if err != nil || qty < 0 {
			result.Errors = append(result.Errors,
				fmt.Sprintf("product %s: invalid quantity '%s'", name, qtyStr))
			continue
		}
		exists, err := s.queries.DishExistsByCode(ctx, number)
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("product %s: failed to validate code '%s': %v", name, number, err))
			continue
		}
		if exists == 0 {
			result.Errors = append(result.Errors,
				fmt.Sprintf("product %s: code '%s' not found", name, number))
			continue
		}

		result.ValidItems = append(result.ValidItems, domain.OrderItem{
			Code:        number,
			ProductName: name,
			Quantity:    qty,
		})
	}

	_, err := s.orderSpec.ValidateBulkOrderUnique(ctx, result.ValidItems)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("validation error: %v", err))
	}

	return result
}

func mapOrderItems(items []sqlc.OrderItem) []domain.OrderItem {
	result := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, domain.OrderItem{
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
		})
	}
	return result
}

func parseRFC3339(value string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (s *OrderService) GetTemplate(ctx context.Context) (string, error) {
	lines := strings.Split(defaultOrderTemplate, "\n")
	result := make([]string, 0, len(lines))
	var missing []string

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			if len(result) > 0 && result[len(result)-1] != "" {
				result = append(result, "")
			}
			continue
		}
		if isTemplateHeader(name) {
			result = append(result, name)
			continue
		}

		products, err := s.queries.GetIikoProductsByName(ctx, name)
		if err != nil {
			return "", fmt.Errorf("get product %q: %w", name, err)
		}

		dishes := make([]sqlc.GetIikoProductsByNameRow, 0, len(products))
		for _, product := range products {
			if product.Type != nil && strings.EqualFold(*product.Type, "DISH") {
				dishes = append(dishes, product)
			}
		}
		if len(dishes) == 0 {
			missing = append(missing, name)
			continue
		}
		sort.Slice(dishes, func(i, j int) bool {
			return dishes[i].Code < dishes[j].Code
		})

		result = append(result, fmt.Sprintf("%s %s 0", dishes[0].Code, dishes[0].Name))
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("template dishes not found: %s", strings.Join(missing, ", "))
	}

	return strings.TrimSpace(strings.Join(result, "\n")), nil
}

func isTemplateHeader(line string) bool {
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
