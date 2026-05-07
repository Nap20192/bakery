package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	orderdomain "bakery/internal/domain/order"
	"bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/helpers"

	"github.com/jackc/pgx/v5"
)

type OrderService struct {
	queries *sqlc.Queries
	domain  *orderdomain.OrderService
}

func NewOrderService(queries *sqlc.Queries) *OrderService {
	return &OrderService{
		queries: queries,
		domain:  orderdomain.NewOrderService(),
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input orderdomain.CreateOrderInput) (orderdomain.Order, error) {
	if len(input.Items) == 0 {
		return orderdomain.Order{}, fmt.Errorf("order must contain items")
	}

	createdAt := s.domain.NormalizeCreatedAt(input.Date)
	day := s.domain.OrderCounterDay(createdAt)

	if err := s.queries.CreateOrderCounterDay(ctx, day); err != nil {
		return orderdomain.Order{}, fmt.Errorf("init order counter: %w", err)
	}
	counter, err := s.queries.NextOrderCounter(ctx, day)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("increment order counter: %w", err)
	}

	number := s.domain.BuildOrderNumber(day, counter)
	row, err := s.queries.CreateOrder(ctx, sqlc.CreateOrderParams{
		Number:    number,
		Location:  input.Location,
		CreatedAt: createdAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("create order: %w", err)
	}

	for _, item := range input.Items {
		var productID *string
		if item.Code != "" {
			product, err := s.queries.GetIikoProductByCode(ctx, item.Code)
			if err != nil && err != pgx.ErrNoRows {
				return orderdomain.Order{}, fmt.Errorf("resolve product by code %s: %w", item.Code, err)
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
			return orderdomain.Order{}, fmt.Errorf("create order item: %w", err)
		}
	}

	return orderdomain.Order{
		ID:        fmt.Sprintf("%d", row.ID),
		Number:    row.Number,
		Location:  row.Location,
		Items:     input.Items,
		CreatedAt: helpers.ParseRFC3339(row.CreatedAt),
	}, nil
}

func (s *OrderService) GetOrderByNumber(ctx context.Context, number string) (orderdomain.Order, error) {
	order, err := s.queries.GetOrderByNumber(ctx, number)
	if err != nil {
		return orderdomain.Order{}, err
	}
	items, err := s.queries.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	return orderdomain.Order{
		ID:        fmt.Sprintf("%d", order.ID),
		Number:    order.Number,
		Location:  order.Location,
		Items:     mapOrderItems(items),
		CreatedAt: helpers.ParseRFC3339(order.CreatedAt),
	}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, limit int32) ([]orderdomain.Order, error) {
	rows, err := s.queries.ListOrders(ctx, limit)
	if err != nil {
		return nil, err
	}
	result := make([]orderdomain.Order, 0, len(rows))
	for _, row := range rows {
		items, err := s.queries.GetOrderItemsByOrderID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, orderdomain.Order{
			ID:        fmt.Sprintf("%d", row.ID),
			Number:    row.Number,
			Location:  row.Location,
			Items:     mapOrderItems(items),
			CreatedAt: helpers.ParseRFC3339(row.CreatedAt),
		})
	}
	return result, nil
}

func (s *OrderService) ValidateBulkOrder(ctx context.Context, order string) orderdomain.BulkOrderValidationResult {
	result := s.domain.ParseBulkOrder(order)

	for _, item := range result.ValidItems {
		exists, err := s.queries.DishExistsByCode(ctx, item.Code)
		if err != nil {
			result.Errors = append(result.Errors, orderdomain.BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: fmt.Sprintf("failed to validate code: %v", err),
			})
			continue
		}
		if exists == 0 {
			result.Errors = append(result.Errors, orderdomain.BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: "product code not found",
			})
		}
	}

	if err := s.domain.ValidateUniqueItems(result.ValidItems); err != nil {
		result.Errors = append(result.Errors, orderdomain.BulkOrderValidationError{
			Message: err.Error(),
		})
	}

	return result
}

func mapOrderItems(items []sqlc.GetOrderItemsByOrderIDRow) []orderdomain.OrderItem {
	result := make([]orderdomain.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, orderdomain.OrderItem{
			Code:        item.ProductCode,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
		})
	}
	return result
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
		if s.domain.IsTemplateHeader(name) {
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
