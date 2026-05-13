package app

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	orderdomain "bakery/internal/domain/order"
	"bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/enum"
	"bakery/internal/pkg/helpers"

	"github.com/jackc/pgx/v5"
)

type OrderService struct {
	queries sqlc.Querier
	domain  *orderdomain.OrderService
}

type ListOrdersInput struct {
	Limit  int32
	Offset int32
}

type ListOrdersResult struct {
	Orders []orderdomain.Order
	Total  int64
	Limit  int32
	Offset int32
}

type UpdateOrderInput struct {
	Number            string
	Items             []orderdomain.OrderItem
	FromDepartmentID  *int64
	ToDepartmentID    *int64
	CreatedByUsername string
	FulfillmentDate   time.Time
}

type EnsureDefaultTemplatesResult struct {
	Created int
	Skipped int
}

func NewOrderService(queries sqlc.Querier) *OrderService {
	return &OrderService{
		queries: queries,
		domain:  orderdomain.NewOrderService(),
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input orderdomain.CreateOrderInput) (orderdomain.Order, error) {
	input.Items = positiveOrderItems(input.Items)
	if len(input.Items) == 0 {
		return orderdomain.Order{}, fmt.Errorf("order must contain items")
	}

	createdAt := s.domain.NormalizeCreatedAt(input.Date)
	fulfillmentDate := s.domain.NormalizeFulfillmentDate(input.FulfillmentDate, createdAt)
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
		Number:            number,
		Location:          input.Location,
		FromDepartmentID:  input.FromDepartmentID,
		ToDepartmentID:    input.ToDepartmentID,
		CreatedAt:         createdAt.Format(time.RFC3339Nano),
		FulfillmentDate:   fulfillmentDate.Format("2006-01-02"),
		CreatedByUsername: strings.TrimSpace(input.CreatedByUsername),
	})
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("create order: %w", err)
	}

	if err := s.createOrderItems(ctx, row.ID, input.Items); err != nil {
		return orderdomain.Order{}, err
	}

	return orderdomain.Order{
		ID:                fmt.Sprintf("%d", row.ID),
		Number:            row.Number,
		Location:          row.Location,
		FromDepartmentID:  row.FromDepartmentID,
		ToDepartmentID:    row.ToDepartmentID,
		CreatedByUsername: row.CreatedByUsername,
		Items:             input.Items,
		CreatedAt:         helpers.ParseRFC3339(row.CreatedAt),
		FulfillmentDate:   parseDate(row.FulfillmentDate),
	}, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, input UpdateOrderInput) (orderdomain.Order, error) {
	input.Items = positiveOrderItems(input.Items)
	input.Number = strings.TrimSpace(input.Number)
	if input.Number == "" {
		return orderdomain.Order{}, fmt.Errorf("order number is required")
	}
	if len(input.Items) == 0 {
		return orderdomain.Order{}, fmt.Errorf("order must contain items")
	}

	existing, err := s.queries.GetOrderByNumber(ctx, input.Number)
	if err != nil {
		return orderdomain.Order{}, err
	}
	if input.FromDepartmentID == nil {
		input.FromDepartmentID = existing.FromDepartmentID
	}
	if input.ToDepartmentID == nil {
		input.ToDepartmentID = existing.ToDepartmentID
	}
	fulfillmentDate := s.domain.NormalizeFulfillmentDate(input.FulfillmentDate, helpers.ParseRFC3339(existing.CreatedAt))
	createdBy := strings.TrimSpace(input.CreatedByUsername)
	if createdBy == "" {
		createdBy = existing.CreatedByUsername
	}

	row, err := s.queries.UpdateOrder(ctx, sqlc.UpdateOrderParams{
		FromDepartmentID:  input.FromDepartmentID,
		ToDepartmentID:    input.ToDepartmentID,
		FulfillmentDate:   fulfillmentDate.Format("2006-01-02"),
		CreatedByUsername: createdBy,
		Number:            input.Number,
	})
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("update order: %w", err)
	}
	if err := s.queries.DeleteOrderItemsByOrderID(ctx, row.ID); err != nil {
		return orderdomain.Order{}, fmt.Errorf("delete order items: %w", err)
	}
	if err := s.createOrderItems(ctx, row.ID, input.Items); err != nil {
		return orderdomain.Order{}, err
	}

	return orderdomain.Order{
		ID:                fmt.Sprintf("%d", row.ID),
		Number:            row.Number,
		Location:          row.Location,
		FromDepartmentID:  row.FromDepartmentID,
		ToDepartmentID:    row.ToDepartmentID,
		CreatedByUsername: row.CreatedByUsername,
		Items:             input.Items,
		CreatedAt:         helpers.ParseRFC3339(row.CreatedAt),
		FulfillmentDate:   parseDate(row.FulfillmentDate),
	}, nil
}

func positiveOrderItems(items []orderdomain.OrderItem) []orderdomain.OrderItem {
	result := make([]orderdomain.OrderItem, 0, len(items))
	for _, item := range items {
		if item.ProductionQuantity() > 0 {
			result = append(result, item)
		}
	}
	return result
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
		ID:                fmt.Sprintf("%d", order.ID),
		Number:            order.Number,
		Location:          order.Location,
		FromDepartmentID:  order.FromDepartmentID,
		ToDepartmentID:    order.ToDepartmentID,
		CreatedByUsername: order.CreatedByUsername,
		Items:             mapOrderItems(items),
		CreatedAt:         helpers.ParseRFC3339(order.CreatedAt),
		FulfillmentDate:   parseDate(order.FulfillmentDate),
	}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, input ListOrdersInput) (ListOrdersResult, error) {
	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	if input.Offset < 0 {
		input.Offset = 0
	}
	total, err := s.queries.CountOrders(ctx)
	if err != nil {
		return ListOrdersResult{}, err
	}
	rows, err := s.queries.ListOrders(ctx, sqlc.ListOrdersParams{
		OrderLimit:  input.Limit,
		OrderOffset: input.Offset,
	})
	if err != nil {
		return ListOrdersResult{}, err
	}
	result := make([]orderdomain.Order, 0, len(rows))
	for _, row := range rows {
		items, err := s.queries.GetOrderItemsByOrderID(ctx, row.ID)
		if err != nil {
			return ListOrdersResult{}, err
		}
		result = append(result, orderdomain.Order{
			ID:                fmt.Sprintf("%d", row.ID),
			Number:            row.Number,
			Location:          row.Location,
			FromDepartmentID:  row.FromDepartmentID,
			ToDepartmentID:    row.ToDepartmentID,
			CreatedByUsername: row.CreatedByUsername,
			Items:             mapOrderItems(items),
			CreatedAt:         helpers.ParseRFC3339(row.CreatedAt),
			FulfillmentDate:   parseDate(row.FulfillmentDate),
		})
	}
	return ListOrdersResult{
		Orders: result,
		Total:  total,
		Limit:  input.Limit,
		Offset: input.Offset,
	}, nil
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

func (s *OrderService) CreateOrderTemplate(ctx context.Context, creatorID *int64, raw string) (orderdomain.OrderTemplate, orderdomain.BulkOrderValidationResult, error) {
	template, validation := s.domain.ParseOrderTemplate(raw)
	if len(validation.Errors) > 0 {
		return orderdomain.OrderTemplate{}, validation, nil
	}
	for _, item := range template.Items {
		exists, err := s.queries.DishExistsByCode(ctx, item.Code)
		if err != nil {
			validation.Errors = append(validation.Errors, orderdomain.BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: fmt.Sprintf("failed to validate code: %v", err),
			})
			continue
		}
		if exists == 0 {
			validation.Errors = append(validation.Errors, orderdomain.BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: "product code not found",
			})
		}
	}
	if len(validation.Errors) > 0 {
		return orderdomain.OrderTemplate{}, validation, nil
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	row, err := s.queries.CreateOrderTemplate(ctx, sqlc.CreateOrderTemplateParams{
		Theme:           template.Theme,
		Name:            template.Name,
		Body:            template.Body,
		CreatedByUserID: creatorID,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return orderdomain.OrderTemplate{}, validation, fmt.Errorf("create order template: %w", err)
	}
	return orderTemplateToDomain(row), validation, nil
}

func (s *OrderService) ListOrderTemplates(ctx context.Context) ([]orderdomain.OrderTemplate, error) {
	rows, err := s.queries.ListOrderTemplates(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]orderdomain.OrderTemplate, 0, len(rows))
	for _, row := range rows {
		result = append(result, orderTemplateToDomain(row))
	}
	return result, nil
}

func (s *OrderService) GetOrderTemplate(ctx context.Context, id int64) (orderdomain.OrderTemplate, error) {
	row, err := s.queries.GetOrderTemplateByID(ctx, id)
	if err != nil {
		return orderdomain.OrderTemplate{}, err
	}
	return orderTemplateToDomain(row), nil
}

func (s *OrderService) EnsureDefaultOrderTemplates(ctx context.Context, path string) (EnsureDefaultTemplatesResult, error) {
	var result EnsureDefaultTemplatesResult
	if strings.TrimSpace(path) == "" {
		path = "templates/dishes.txt"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, fmt.Errorf("read default templates: %w", err)
	}

	defaults := parseDefaultOrderTemplates(string(data))
	if len(defaults) == 0 {
		return result, nil
	}

	existingRows, err := s.queries.ListOrderTemplates(ctx)
	if err != nil {
		return result, fmt.Errorf("list templates: %w", err)
	}
	existing := make(map[string]struct{}, len(existingRows))
	for _, row := range existingRows {
		existing[normalizeTemplateName(row.Name)] = struct{}{}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, template := range defaults {
		key := normalizeTemplateName(template.Name)
		if _, ok := existing[key]; ok {
			result.Skipped++
			continue
		}
		if _, err := s.queries.CreateOrderTemplate(ctx, sqlc.CreateOrderTemplateParams{
			Theme:           template.Theme,
			Name:            template.Name,
			Body:            template.Body,
			CreatedByUserID: nil,
			CreatedAt:       now,
			UpdatedAt:       now,
		}); err != nil {
			return result, fmt.Errorf("create default template %q: %w", template.Name, err)
		}
		existing[key] = struct{}{}
		result.Created++
	}
	return result, nil
}

func (s *OrderService) createOrderItems(ctx context.Context, orderID int64, items []orderdomain.OrderItem) error {
	for _, item := range items {
		if item.ProductionQuantity() <= 0 {
			continue
		}
		var productID *string
		if item.Code != "" {
			product, err := s.queries.GetIikoProductByCode(ctx, item.Code)
			if err != nil && err != pgx.ErrNoRows {
				return fmt.Errorf("resolve product by code %s: %w", item.Code, err)
			}
			if err == nil {
				productID = &product.ID
			}
		}
		if _, err := s.queries.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
			OrderID:          orderID,
			IikoProductID:    productID,
			ProductName:      item.ProductName,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		}); err != nil {
			return fmt.Errorf("create order item: %w", err)
		}
	}
	return nil
}

func mapOrderItems(items []sqlc.GetOrderItemsByOrderIDRow) []orderdomain.OrderItem {
	result := make([]orderdomain.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, orderdomain.OrderItem{
			Code:             item.ProductCode,
			ProductName:      item.ProductName,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		})
	}
	return result
}

func orderTemplateToDomain(row sqlc.OrderTemplate) orderdomain.OrderTemplate {
	return orderdomain.OrderTemplate{
		ID:              row.ID,
		Theme:           row.Theme,
		Name:            row.Name,
		Body:            row.Body,
		CreatedByUserID: row.CreatedByUserID,
		CreatedAt:       helpers.ParseRFC3339(row.CreatedAt),
		UpdatedAt:       helpers.ParseRFC3339(row.UpdatedAt),
	}
}

func parseDate(value string) time.Time {
	t, err := time.Parse("2006-01-02", value)
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
			if product.Type != nil && enum.IsIikoProductType(*product.Type, enum.IikoProductTypeDish) {
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
