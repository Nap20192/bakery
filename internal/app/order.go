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
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderService struct {
	queries   sqlc.Querier
	txQueries *sqlc.Queries
	db        *pgxpool.Pool
	domain    *orderdomain.OrderService
}

type ListOrdersInput struct {
	Limit            int32
	Offset           int32
	FromDepartmentID *int64
	FulfillmentDate  time.Time
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

func NewOrderServiceWithDB(queries *sqlc.Queries, db *pgxpool.Pool) *OrderService {
	svc := NewOrderService(queries)
	svc.txQueries = queries
	svc.db = db
	return svc
}

func (s *OrderService) CreateOrder(ctx context.Context, input orderdomain.CreateOrderInput) (orderdomain.Order, error) {
	input.Items = positiveOrderItems(input.Items)
	if len(input.Items) == 0 {
		return orderdomain.Order{}, fmt.Errorf("order must contain items")
	}

	createdAt := s.domain.NormalizeCreatedAt(input.Date)
	fulfillmentDate := s.domain.NormalizeFulfillmentDate(input.FulfillmentDate, createdAt)
	day := s.domain.OrderCounterDay(createdAt)
	shop, err := s.orderShopDepartment(ctx, input.FromDepartmentID)
	if err != nil {
		return orderdomain.Order{}, err
	}

	if s.db != nil && s.txQueries != nil {
		return s.createOrderTx(ctx, input, shop, createdAt, fulfillmentDate, day)
	}
	return s.createOrderWithQueries(ctx, s.queries, input, shop, createdAt, fulfillmentDate, day)
}

func (s *OrderService) createOrderTx(
	ctx context.Context,
	input orderdomain.CreateOrderInput,
	shop sqlc.Department,
	createdAt time.Time,
	fulfillmentDate time.Time,
	day string,
) (orderdomain.Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("begin order tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	order, err := s.createOrderWithQueries(ctx, s.txQueries.WithTx(tx), input, shop, createdAt, fulfillmentDate, day)
	if err != nil {
		return orderdomain.Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return orderdomain.Order{}, fmt.Errorf("commit order tx: %w", err)
	}
	committed = true
	return order, nil
}

func (s *OrderService) createOrderWithQueries(
	ctx context.Context,
	q sqlc.Querier,
	input orderdomain.CreateOrderInput,
	shop sqlc.Department,
	createdAt time.Time,
	fulfillmentDate time.Time,
	day string,
) (orderdomain.Order, error) {
	if err := q.CreateOrderCounterDay(ctx, day); err != nil {
		return orderdomain.Order{}, fmt.Errorf("init order counter: %w", err)
	}
	counter, err := q.NextOrderCounter(ctx, day)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("increment order counter: %w", err)
	}

	number := s.domain.BuildOrderNumber(shop.Code, shop.Name, createdAt, counter)
	row, err := q.CreateOrder(ctx, sqlc.CreateOrderParams{
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

	if err := s.createOrderItems(ctx, q, row.ID, input.Items); err != nil {
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

func (s *OrderService) orderShopDepartment(ctx context.Context, departmentID *int64) (sqlc.Department, error) {
	if departmentID == nil {
		return sqlc.Department{}, fmt.Errorf("order shop department is required")
	}
	department, err := s.queries.GetDepartmentByID(ctx, *departmentID)
	if err != nil {
		return sqlc.Department{}, fmt.Errorf("get order shop department: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(department.Type), string(enum.DepartmentTypeShop)) {
		return sqlc.Department{}, fmt.Errorf("order can be created only from shop department")
	}
	if strings.TrimSpace(department.Code) == "" && strings.TrimSpace(department.Name) == "" {
		return sqlc.Department{}, fmt.Errorf("order shop department code or name is required")
	}
	return department, nil
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
	existingItemsRows, err := s.queries.GetOrderItemsByOrderID(ctx, existing.ID)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("get existing order items: %w", err)
	}
	existingItems := mapOrderItems(existingItemsRows)
	historyItems := diffOrderItems(existingItems, input.Items)
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
	if err := s.createOrderItems(ctx, s.queries, row.ID, input.Items); err != nil {
		return orderdomain.Order{}, err
	}
	if len(historyItems) > 0 {
		if err := s.createOrderHistory(ctx, row.ID, createdBy, historyItems); err != nil {
			return orderdomain.Order{}, err
		}
	}
	history, err := s.listOrderHistory(ctx, row.ID)
	if err != nil {
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
		History:           history,
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
	history, err := s.listOrderHistory(ctx, order.ID)
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
		History:           history,
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
	var fulfillmentDate *string
	if !input.FulfillmentDate.IsZero() {
		value := input.FulfillmentDate.Format("2006-01-02")
		fulfillmentDate = &value
	}
	total, err := s.queries.CountOrders(ctx, sqlc.CountOrdersParams{
		FromDepartmentID: input.FromDepartmentID,
		FulfillmentDate:  fulfillmentDate,
	})
	if err != nil {
		return ListOrdersResult{}, err
	}
	rows, err := s.queries.ListOrders(ctx, sqlc.ListOrdersParams{
		FromDepartmentID: input.FromDepartmentID,
		FulfillmentDate:  fulfillmentDate,
		OrderLimit:       input.Limit,
		OrderOffset:      input.Offset,
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
				Message: "Не удалось проверить код продукта. Попробуйте позже.",
			})
			continue
		}
		if exists == 0 {
			result.Errors = append(result.Errors, orderdomain.BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: "Код продукта не найден. Проверьте код в шаблоне или заказе.",
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
				Message: "Не удалось проверить код продукта. Попробуйте позже.",
			})
			continue
		}
		if exists == 0 {
			validation.Errors = append(validation.Errors, orderdomain.BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: "Код продукта не найден. Проверьте код в шаблоне или заказе.",
			})
		}
	}
	if len(validation.Errors) > 0 {
		return orderdomain.OrderTemplate{}, validation, nil
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	row, err := s.queries.CreateOrderTemplate(ctx, sqlc.CreateOrderTemplateParams{
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

func (s *OrderService) CombinedOrderTemplate(ctx context.Context) (string, error) {
	templates, err := s.ListOrderTemplates(ctx)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(templates))
	for _, template := range templates {
		body := strings.TrimSpace(template.Body)
		if body == "" {
			continue
		}
		parts = append(parts, body)
	}
	return strings.Join(parts, "\n\n"), nil
}

func (s *OrderService) GetOrderTemplate(ctx context.Context, id int64) (orderdomain.OrderTemplate, error) {
	row, err := s.queries.GetOrderTemplateByID(ctx, id)
	if err != nil {
		return orderdomain.OrderTemplate{}, err
	}
	return orderTemplateToDomain(row), nil
}

func (s *OrderService) DeleteOrderTemplate(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("template id is required")
	}
	if _, err := s.queries.GetOrderTemplateByID(ctx, id); err != nil {
		return fmt.Errorf("get order template: %w", err)
	}
	if err := s.queries.DeleteOrderTemplateByID(ctx, id); err != nil {
		return fmt.Errorf("delete order template: %w", err)
	}
	return nil
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

func (s *OrderService) createOrderItems(ctx context.Context, q sqlc.Querier, orderID int64, items []orderdomain.OrderItem) error {
	for _, item := range items {
		if item.ProductionQuantity() <= 0 {
			continue
		}
		var productID *string
		if item.Code != "" {
			product, err := q.GetIikoProductByCode(ctx, item.Code)
			if err != nil && err != pgx.ErrNoRows {
				return fmt.Errorf("resolve product by code %s: %w", item.Code, err)
			}
			if err == nil {
				productID = &product.ID
			}
		}
		if _, err := q.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
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

func (s *OrderService) createOrderHistory(ctx context.Context, orderID int64, changedBy string, items []orderdomain.OrderHistoryItem) error {
	history, err := s.queries.CreateOrderHistory(ctx, sqlc.CreateOrderHistoryParams{
		OrderID:           orderID,
		ChangedByUsername: strings.TrimSpace(changedBy),
		ChangedAt:         time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("create order history: %w", err)
	}
	for _, item := range items {
		if _, err := s.queries.CreateOrderHistoryItem(ctx, sqlc.CreateOrderHistoryItemParams{
			HistoryID:           history.ID,
			ChangeType:          item.ChangeType,
			ProductCode:         item.ProductCode,
			ProductName:         item.ProductName,
			OldQuantity:         item.OldQuantity,
			NewQuantity:         item.NewQuantity,
			OldReservedQuantity: item.OldReservedQuantity,
			NewReservedQuantity: item.NewReservedQuantity,
		}); err != nil {
			return fmt.Errorf("create order history item: %w", err)
		}
	}
	return nil
}

func (s *OrderService) listOrderHistory(ctx context.Context, orderID int64) ([]orderdomain.OrderHistory, error) {
	rows, err := s.queries.ListOrderHistoryByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	result := make([]orderdomain.OrderHistory, 0, len(rows))
	for _, row := range rows {
		itemRows, err := s.queries.ListOrderHistoryItemsByHistoryID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		items := make([]orderdomain.OrderHistoryItem, 0, len(itemRows))
		for _, item := range itemRows {
			items = append(items, orderdomain.OrderHistoryItem{
				ChangeType:          item.ChangeType,
				ProductCode:         item.ProductCode,
				ProductName:         item.ProductName,
				OldQuantity:         item.OldQuantity,
				NewQuantity:         item.NewQuantity,
				OldReservedQuantity: item.OldReservedQuantity,
				NewReservedQuantity: item.NewReservedQuantity,
			})
		}
		result = append(result, orderdomain.OrderHistory{
			ID:                row.ID,
			ChangedByUsername: row.ChangedByUsername,
			ChangedAt:         helpers.ParseRFC3339(row.ChangedAt),
			Items:             items,
		})
	}
	return result, nil
}

func diffOrderItems(oldItems []orderdomain.OrderItem, newItems []orderdomain.OrderItem) []orderdomain.OrderHistoryItem {
	oldByCode := make(map[string]orderdomain.OrderItem, len(oldItems))
	newByCode := make(map[string]orderdomain.OrderItem, len(newItems))
	for _, item := range oldItems {
		oldByCode[item.Code] = item
	}
	for _, item := range newItems {
		newByCode[item.Code] = item
	}

	result := make([]orderdomain.OrderHistoryItem, 0)
	for _, item := range newItems {
		old, ok := oldByCode[item.Code]
		if !ok {
			result = append(result, orderHistoryItem("added", orderdomain.OrderItem{}, item))
			continue
		}
		if old.Quantity != item.Quantity || old.ReservedQuantity != item.ReservedQuantity || strings.TrimSpace(old.ProductName) != strings.TrimSpace(item.ProductName) {
			result = append(result, orderHistoryItem("updated", old, item))
		}
	}
	for _, item := range oldItems {
		if _, ok := newByCode[item.Code]; !ok {
			result = append(result, orderHistoryItem("removed", item, orderdomain.OrderItem{}))
		}
	}
	return result
}

func orderHistoryItem(changeType string, oldItem orderdomain.OrderItem, newItem orderdomain.OrderItem) orderdomain.OrderHistoryItem {
	productCode := newItem.Code
	if productCode == "" {
		productCode = oldItem.Code
	}
	productName := strings.TrimSpace(newItem.ProductName)
	if productName == "" {
		productName = strings.TrimSpace(oldItem.ProductName)
	}
	item := orderdomain.OrderHistoryItem{
		ChangeType:  changeType,
		ProductCode: productCode,
		ProductName: productName,
	}
	if changeType != "added" {
		item.OldQuantity = float64Ptr(oldItem.Quantity)
		item.OldReservedQuantity = float64Ptr(oldItem.ReservedQuantity)
	}
	if changeType != "removed" {
		item.NewQuantity = float64Ptr(newItem.Quantity)
		item.NewReservedQuantity = float64Ptr(newItem.ReservedQuantity)
	}
	return item
}

func float64Ptr(value float64) *float64 {
	return &value
}

func orderTemplateToDomain(row sqlc.OrderTemplate) orderdomain.OrderTemplate {
	return orderdomain.OrderTemplate{
		ID:              row.ID,
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
