package orderuc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"bakery/internal/pkg/apperr"
	"bakery/internal/pkg/enum"
	orderdomain "bakery/internal/services/order/domain"
)

var (
	ErrDishCatalogItemNotFound  = apperr.NotFound("order.dish_not_found", "Блюдо не найдено в каталоге.")
	ErrDishCatalogItemAmbiguous = apperr.Conflict("order.dish_ambiguous", "Найдено несколько блюд с таким названием.")
	ErrFulfillmentDateInPast    = apperr.Invalid("order.fulfillment_date_in_past", "Дата выполнения не может быть в прошлом.")
)

// Service is the order use-case implementation. It depends only on the
// Repository port and the pure domain service — never on infrastructure.
// Domain events are persisted to the transactional outbox by the repository
// (atomically with the write) and published by a separate relay.
type Service struct {
	repo   Repository
	domain *orderdomain.OrderService
}

// Compile-time assurance that Service satisfies the boundary contract.
var _ UseCase = (*Service)(nil)

func NewService(repo Repository) *Service {
	return &Service{
		repo:   repo,
		domain: orderdomain.NewOrderService(),
	}
}

func (s *Service) CreateOrder(ctx context.Context, input orderdomain.CreateOrderInput) (orderdomain.Order, error) {
	if s.repo == nil {
		return orderdomain.Order{}, fmt.Errorf("missing order repository")
	}
	resolvedItems, err := s.resolveOrderItems(ctx, input.Items)
	if err != nil {
		return orderdomain.Order{}, err
	}
	input.Items = positiveOrderItems(resolvedItems)
	if len(input.Items) == 0 {
		return orderdomain.Order{}, fmt.Errorf("order must contain items")
	}

	createdAt := s.domain.NormalizeCreatedAt(input.Date)
	fulfillmentDate := s.domain.NormalizeFulfillmentDate(input.FulfillmentDate, createdAt)
	if err := validateFulfillmentDateNotPast(fulfillmentDate, createdAt); err != nil {
		return orderdomain.Order{}, err
	}
	shop, err := s.orderShopDepartment(ctx, input.FromDepartmentID)
	if err != nil {
		return orderdomain.Order{}, err
	}

	order, err := s.repo.CreateOrder(ctx, CreateOrderRepositoryInput{
		Input:           input,
		Shop:            shop,
		CreatedAt:       createdAt,
		FulfillmentDate: fulfillmentDate,
		CounterDay:      s.domain.OrderCounterDay(createdAt),
	})
	if err != nil {
		return orderdomain.Order{}, err
	}
	return order, nil
}

func (s *Service) UpdateOrder(ctx context.Context, input UpdateOrderInput) (orderdomain.Order, error) {
	if s.repo == nil {
		return orderdomain.Order{}, fmt.Errorf("missing order repository")
	}
	resolvedItems, err := s.resolveOrderItems(ctx, input.Items)
	if err != nil {
		return orderdomain.Order{}, err
	}
	input.Items = positiveOrderItems(resolvedItems)
	input.Number = strings.TrimSpace(input.Number)
	if input.Number == "" {
		return orderdomain.Order{}, fmt.Errorf("order number is required")
	}
	if len(input.Items) == 0 {
		return orderdomain.Order{}, fmt.Errorf("order must contain items")
	}

	existing, err := s.repo.GetOrderByNumber(ctx, input.Number)
	if err != nil {
		return orderdomain.Order{}, err
	}
	historyItems := diffOrderItems(existing.Items, input.Items)
	if input.FromDepartmentID == nil {
		input.FromDepartmentID = existing.FromDepartmentID
	}
	if input.ToDepartmentID == nil {
		input.ToDepartmentID = existing.ToDepartmentID
	}
	fulfillmentDate := s.domain.NormalizeFulfillmentDate(input.FulfillmentDate, existing.CreatedAt)
	if err := validateFulfillmentDateNotPast(fulfillmentDate, time.Now().UTC()); err != nil {
		return orderdomain.Order{}, err
	}
	createdBy := strings.TrimSpace(input.CreatedByUsername)
	if createdBy == "" {
		createdBy = existing.CreatedByUsername
	}

	order, err := s.repo.UpdateOrder(ctx, UpdateOrderRepositoryInput{
		Number:            input.Number,
		Items:             input.Items,
		FromDepartmentID:  input.FromDepartmentID,
		ToDepartmentID:    input.ToDepartmentID,
		CreatedByUsername: createdBy,
		FulfillmentDate:   fulfillmentDate,
		Comments:          input.Comments,
		HistoryItems:      historyItems,
	})
	if err != nil {
		return orderdomain.Order{}, err
	}
	return order, nil
}

func validateFulfillmentDateNotPast(fulfillmentDate time.Time, now time.Time) error {
	today := dateOnly(now)
	if dateOnly(fulfillmentDate).Before(today) {
		return ErrFulfillmentDateInPast
	}
	return nil
}

func dateOnly(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) GetOrderByNumber(ctx context.Context, number string) (orderdomain.Order, error) {
	if s.repo == nil {
		return orderdomain.Order{}, fmt.Errorf("missing order repository")
	}
	return s.repo.GetOrderByNumber(ctx, strings.TrimSpace(number))
}

func (s *Service) ListOrders(ctx context.Context, input ListOrdersInput) (ListOrdersResult, error) {
	if s.repo == nil {
		return ListOrdersResult{}, fmt.Errorf("missing order repository")
	}
	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	if input.Offset < 0 {
		input.Offset = 0
	}
	return s.repo.ListOrders(ctx, input)
}

func (s *Service) ValidateBulkOrder(ctx context.Context, order string) orderdomain.BulkOrderValidationResult {
	result := s.domain.ParseBulkOrder(order)
	result.ValidItems = s.resolveValidOrderItems(ctx, result.ValidItems, &result)

	for _, item := range result.ValidItems {
		exists, err := s.repo.DishExistsByCode(ctx, item.Code)
		if err != nil {
			result.Errors = append(result.Errors, orderdomain.BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: "Не удалось проверить код продукта. Попробуйте позже.",
			})
			continue
		}
		if !exists {
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

func (s *Service) ListDishCatalog(ctx context.Context) ([]orderdomain.DishCatalogItem, error) {
	rows, err := s.repo.ListDishCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dish catalog items: %w", err)
	}
	items := make([]orderdomain.DishCatalogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, orderdomain.DishCatalogItem{
			Code:      row.Code,
			Name:      row.Name,
			Theme:     row.Theme,
			SortOrder: row.SortOrder,
		})
	}
	return items, nil
}

// AddDishCatalogItem inserts (or updates) a dish in the catalog. The code is
// derived from the name so admin-added dishes resolve by name like the rest.
func (s *Service) AddDishCatalogItem(ctx context.Context, name, theme string) (orderdomain.DishCatalogItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return orderdomain.DishCatalogItem{}, apperr.Invalid("order.dish_name_required", "Укажите название блюда.")
	}
	theme = strings.TrimSpace(theme)
	code := "custom:" + strings.ToLower(strings.Join(strings.Fields(name), " "))
	if err := s.repo.UpsertDishCatalogItem(ctx, DishCatalogItem{Code: code, Name: name, Theme: theme}); err != nil {
		return orderdomain.DishCatalogItem{}, fmt.Errorf("add dish catalog item: %w", err)
	}
	return orderdomain.DishCatalogItem{Code: code, Name: name, Theme: theme}, nil
}

// DeleteDishCatalogItem removes a dish from the catalog by code.
func (s *Service) DeleteDishCatalogItem(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return apperr.Invalid("order.dish_code_required", "Укажите блюдо.")
	}
	return s.repo.DeleteDishCatalogItem(ctx, code)
}

func (s *Service) ListOrderTemplates(ctx context.Context) ([]orderdomain.OrderTemplate, error) {
	rows, err := s.repo.ListDishCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dish catalog items: %w", err)
	}
	return dishCatalogTemplates(rows), nil
}

func (s *Service) CombinedOrderTemplate(ctx context.Context) (string, error) {
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

func (s *Service) GetOrderTemplate(ctx context.Context, theme string) (orderdomain.OrderTemplate, error) {
	theme = normalizeTemplateName(theme)
	if theme == "" {
		return orderdomain.OrderTemplate{}, fmt.Errorf("template name is required")
	}
	templates, err := s.ListOrderTemplates(ctx)
	if err != nil {
		return orderdomain.OrderTemplate{}, err
	}
	for _, template := range templates {
		if normalizeTemplateName(template.Name) == theme {
			return template, nil
		}
	}
	return orderdomain.OrderTemplate{}, fmt.Errorf("template %q not found", theme)
}

func (s *Service) GetTemplate(ctx context.Context) (string, error) {
	return s.CombinedOrderTemplate(ctx)
}

func (s *Service) EnsureDefaultOrderTemplates(ctx context.Context, path string) (EnsureDefaultTemplatesResult, error) {
	var result EnsureDefaultTemplatesResult
	if strings.TrimSpace(path) == "" {
		path = "templates/dishes.txt"
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is a configured local template file.
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, fmt.Errorf("read default templates: %w", err)
	}

	for _, item := range parseDefaultDishCatalogItems(string(data)) {
		if err := s.repo.UpsertDishCatalogItem(ctx, item); err != nil {
			return result, fmt.Errorf("upsert dish catalog item %s: %w", item.Code, err)
		}
		result.CatalogItems++
	}
	return result, nil
}

func (s *Service) DeleteOrdersOlderThan(ctx context.Context, now time.Time, retention time.Duration) (int64, error) {
	if retention <= 0 {
		return 0, fmt.Errorf("order retention must be positive")
	}
	cutoff := now.UTC().Add(-retention)
	count, err := s.repo.DeleteOrdersOlderThan(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old orders: %w", err)
	}
	return count, nil
}

func (s *Service) RunCleanupTicker(ctx context.Context, interval, retention time.Duration) error {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if retention <= 0 {
		retention = 31 * 24 * time.Hour
	}

	run := func() {
		deleted, err := s.DeleteOrdersOlderThan(ctx, time.Now(), retention)
		if err != nil {
			slog.ErrorContext(ctx, "old orders cleanup failed", "error", err)
			return
		}
		slog.InfoContext(ctx, "old orders cleanup finished", "deleted", deleted, "retention", retention.String())
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			run()
		}
	}
}

func (s *Service) orderShopDepartment(ctx context.Context, departmentID *int64) (Department, error) {
	if departmentID == nil {
		return Department{}, fmt.Errorf("order shop department is required")
	}
	department, err := s.repo.GetDepartmentByID(ctx, *departmentID)
	if err != nil {
		return Department{}, fmt.Errorf("get order shop department: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(department.Type), string(enum.DepartmentTypeShop)) {
		return Department{}, fmt.Errorf("order can be created only from shop department")
	}
	if strings.TrimSpace(department.Code) == "" && strings.TrimSpace(department.Name) == "" {
		return Department{}, fmt.Errorf("order shop department code or name is required")
	}
	return department, nil
}

func (s *Service) resolveValidOrderItems(
	ctx context.Context,
	items []orderdomain.OrderItem,
	result *orderdomain.BulkOrderValidationResult,
) []orderdomain.OrderItem {
	resolvedItems := make([]orderdomain.OrderItem, 0, len(items))
	for _, item := range items {
		resolved, err := s.resolveOrderItem(ctx, item)
		if err != nil {
			result.Errors = append(result.Errors, orderdomain.BulkOrderValidationError{
				Code:    item.Code,
				Name:    item.ProductName,
				Message: dishCatalogValidationMessage(item.ProductName, err),
			})
			continue
		}
		resolvedItems = append(resolvedItems, resolved)
	}
	return resolvedItems
}

func (s *Service) resolveOrderItems(ctx context.Context, items []orderdomain.OrderItem) ([]orderdomain.OrderItem, error) {
	resolvedItems := make([]orderdomain.OrderItem, 0, len(items))
	for _, item := range items {
		resolved, err := s.resolveOrderItem(ctx, item)
		if err != nil {
			return nil, err
		}
		resolvedItems = append(resolvedItems, resolved)
	}
	return resolvedItems, nil
}

func (s *Service) resolveOrderItem(ctx context.Context, item orderdomain.OrderItem) (orderdomain.OrderItem, error) {
	item.Code = strings.TrimSpace(item.Code)
	item.ProductName = strings.TrimSpace(item.ProductName)
	if item.Code != "" {
		return item, nil
	}
	if item.ProductName == "" {
		return item, ErrDishCatalogItemNotFound
	}

	catalogItem, err := s.repo.ResolveDishCatalogItem(ctx, item.ProductName)
	if err != nil {
		return item, err
	}
	item.Code = catalogItem.Code
	item.ProductName = catalogItem.Name
	return item, nil
}

func dishCatalogValidationMessage(name string, err error) string {
	switch {
	case errors.Is(err, ErrDishCatalogItemNotFound):
		return fmt.Sprintf("Блюдо %q не найдено в справочнике. Проверьте название или выберите позицию из шаблона.", strings.TrimSpace(name))
	case errors.Is(err, ErrDishCatalogItemAmbiguous):
		return fmt.Sprintf("Блюдо %q найдено несколько раз. Уточните название или отправьте строку с кодом.", strings.TrimSpace(name))
	default:
		return "Не удалось проверить блюдо по справочнику. Попробуйте позже."
	}
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

func diffOrderItems(oldItems, newItems []orderdomain.OrderItem) []orderdomain.OrderHistoryItem {
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

func orderHistoryItem(changeType string, oldItem, newItem orderdomain.OrderItem) orderdomain.OrderHistoryItem {
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

func dishCatalogTemplates(items []DishCatalogItem) []orderdomain.OrderTemplate {
	const fallbackTheme = "БЕЗ ТЕМЫ"

	order := make([]string, 0)
	linesByTheme := make(map[string][]string)
	seenThemes := make(map[string]string)
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		theme := strings.TrimSpace(item.Theme)
		if theme == "" {
			theme = fallbackTheme
		}
		key := normalizeTemplateName(theme)
		if _, ok := seenThemes[key]; !ok {
			seenThemes[key] = theme
			order = append(order, key)
		}
		linesByTheme[key] = append(linesByTheme[key], fmt.Sprintf("%s 0", name))
	}

	result := make([]orderdomain.OrderTemplate, 0, len(order))
	for _, key := range order {
		theme := seenThemes[key]
		lines := make([]string, 0, len(linesByTheme[key])+1)
		lines = append(lines, theme)
		lines = append(lines, linesByTheme[key]...)
		result = append(result, orderdomain.OrderTemplate{
			ID:   int64(len(result) + 1),
			Name: theme,
			Body: strings.Join(lines, "\n"),
		})
	}
	return result
}

func normalizeTemplateName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

func parseDefaultDishCatalogItems(raw string) []DishCatalogItem {
	var items []DishCatalogItem
	currentTheme := ""
	spec := orderdomain.NewOrderSpec()

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if orderdomain.IsTemplateHeaderLine(line) {
			currentTheme = line
			continue
		}
		if currentTheme == "" {
			continue
		}
		parsed := orderdomain.ParseOrderLine(orderdomain.BulkOrderLine{Raw: line})
		if parsed.Code == "" || parsed.Name == "" || !spec.Quantity.IsValid(parsed) {
			continue
		}
		items = append(items, DishCatalogItem{
			Code:      parsed.Code,
			Name:      parsed.Name,
			Theme:     currentTheme,
			SortOrder: int64(len(items) + 1),
		})
	}

	return items
}
