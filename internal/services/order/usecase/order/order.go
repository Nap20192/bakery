package orderuc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
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
	ErrOrderCancelled           = apperr.Conflict("order.cancelled", "Отменённый заказ нельзя изменить. Сначала восстановите его.")
	ErrCategoryRequired         = apperr.Invalid("order.category_required", "Выберите тип заявки.")
	ErrCategoryNotFound         = apperr.NotFound("order.category_not_found", "Тип заявки не найден.")
	ErrCategoryHasDishes        = apperr.Conflict("order.category_has_dishes", "У типа заявки есть блюда. Сначала перенесите их в другой тип.")
	ErrProductionOrderNotFound  = apperr.NotFound("order.production_order_not_found", "Заказ для отработки не найден.")
	ErrProductionSheetNotFound  = apperr.NotFound("order.production_sheet_not_found", "Отработка не найдена.")
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

// validatedOrderWrite is the outcome of validateOrderWrite: resolved items
// plus everything CreateOrder and SaveOrderDraft both need to persist their
// write (source department, category, normalized fulfillment date).
type validatedOrderWrite struct {
	Items           []orderdomain.OrderItem
	FulfillmentDate time.Time
	Source          Department
	Category        orderdomain.OrderCategory
}

// validateOrderWrite runs the validation shared by CreateOrder and
// SaveOrderDraft: resolve items against the dish catalog, drop zero-quantity
// lines, reject a past fulfillment date, and check the source department and
// category both exist. now is the reference point for "not in the past" —
// CreateOrder passes the order's createdAt, SaveOrderDraft passes time.Now().
func (s *Service) validateOrderWrite(
	ctx context.Context,
	items []orderdomain.OrderItem,
	categoryID int64,
	fromDepartmentID *int64,
	rawFulfillmentDate time.Time,
	now time.Time,
) (validatedOrderWrite, error) {
	resolvedItems, err := s.resolveOrderItems(ctx, items)
	if err != nil {
		return validatedOrderWrite{}, err
	}
	resolvedItems = positiveOrderItems(resolvedItems)
	if len(resolvedItems) == 0 {
		return validatedOrderWrite{}, fmt.Errorf("order must contain items")
	}

	fulfillmentDate := s.domain.NormalizeFulfillmentDate(rawFulfillmentDate, now)
	if err := validateFulfillmentDateNotPast(fulfillmentDate, now); err != nil {
		return validatedOrderWrite{}, err
	}
	source, err := s.orderSourceDepartment(ctx, fromDepartmentID)
	if err != nil {
		return validatedOrderWrite{}, err
	}
	if categoryID <= 0 {
		return validatedOrderWrite{}, ErrCategoryRequired
	}
	category, err := s.repo.GetOrderCategoryByID(ctx, categoryID)
	if err != nil {
		return validatedOrderWrite{}, ErrCategoryNotFound
	}
	return validatedOrderWrite{
		Items:           resolvedItems,
		FulfillmentDate: fulfillmentDate,
		Source:          source,
		Category:        category,
	}, nil
}

func (s *Service) CreateOrder(ctx context.Context, input orderdomain.CreateOrderInput) (orderdomain.Order, error) {
	if s.repo == nil {
		return orderdomain.Order{}, fmt.Errorf("missing order repository")
	}
	createdAt := s.domain.NormalizeCreatedAt(input.Date)
	validated, err := s.validateOrderWrite(ctx, input.Items, input.CategoryID, input.FromDepartmentID, input.FulfillmentDate, createdAt)
	if err != nil {
		return orderdomain.Order{}, err
	}
	input.Items = validated.Items

	order, err := s.repo.CreateOrder(ctx, CreateOrderRepositoryInput{
		Input:           input,
		Source:          validated.Source,
		Category:        validated.Category,
		CreatedAt:       createdAt,
		FulfillmentDate: validated.FulfillmentDate,
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
	if existing.Cancelled {
		return orderdomain.Order{}, ErrOrderCancelled
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
	order, err := s.repo.UpdateOrder(ctx, UpdateOrderRepositoryInput{
		Number:            input.Number,
		Items:             input.Items,
		FromDepartmentID:  input.FromDepartmentID,
		ToDepartmentID:    input.ToDepartmentID,
		ChangedByUsername: strings.TrimSpace(input.CreatedByUsername),
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

func (s *Service) SetOrderFavorite(ctx context.Context, number string, favorite bool) (orderdomain.Order, error) {
	number = strings.TrimSpace(number)
	if number == "" {
		return orderdomain.Order{}, apperr.Invalid("order.number_required", "Укажите номер заказа.")
	}
	return s.repo.SetOrderFavorite(ctx, number, favorite)
}

// CancelOrder soft-cancels an order. Already-cancelled orders are returned
// unchanged (idempotent), so a double click is harmless.
func (s *Service) CancelOrder(ctx context.Context, number, byUsername string) (orderdomain.Order, error) {
	number = strings.TrimSpace(number)
	if number == "" {
		return orderdomain.Order{}, apperr.Invalid("order.number_required", "Укажите номер заказа.")
	}
	existing, err := s.repo.GetOrderByNumber(ctx, number)
	if err != nil {
		return orderdomain.Order{}, err
	}
	if existing.Cancelled {
		return existing, nil
	}
	return s.repo.CancelOrder(ctx, number, strings.TrimSpace(byUsername))
}

// CreateProductionSheet создаёт документ отработки: фиксирует партию
// (выбранные заказы) и отклонения факта от заявки. Заказы не изменяются —
// факт декорирует их при чтении; наружу уходят события order.produced для
// заказов, чей видимый факт изменился.
func (s *Service) CreateProductionSheet(ctx context.Context, input RecordProductionInput) (orderdomain.ProductionSheet, error) {
	orders, err := s.validateProductionInput(ctx, input)
	if err != nil {
		return orderdomain.ProductionSheet{}, err
	}
	return s.repo.SaveProductionSheet(ctx, SaveProductionSheetInput{
		ProducedByUsername: strings.TrimSpace(input.ProducedByUsername),
		Orders:             orders,
	})
}

// UpdateProductionSheet заменяет партию и отклонения существующего документа.
// Значение, равное заявке, убирает строку отклонения; лист живёт, пока его
// не удалят явно — даже если отклонений не осталось (партия зафиксирована).
func (s *Service) UpdateProductionSheet(ctx context.Context, id int64, input RecordProductionInput) (orderdomain.ProductionSheet, error) {
	if id <= 0 {
		return orderdomain.ProductionSheet{}, ErrProductionSheetNotFound
	}
	if _, err := s.repo.GetProductionSheet(ctx, id); err != nil {
		return orderdomain.ProductionSheet{}, ErrProductionSheetNotFound
	}
	orders, err := s.validateProductionInput(ctx, input)
	if err != nil {
		return orderdomain.ProductionSheet{}, err
	}
	return s.repo.SaveProductionSheet(ctx, SaveProductionSheetInput{
		SheetID:            id,
		ProducedByUsername: strings.TrimSpace(input.ProducedByUsername),
		Orders:             orders,
	})
}

// DeleteProductionSheet удаляет документ отработки; факт в затронутых заказах
// пересчитывается по оставшимся листам журнала.
func (s *Service) DeleteProductionSheet(ctx context.Context, id int64, byUsername string) error {
	if id <= 0 {
		return ErrProductionSheetNotFound
	}
	if _, err := s.repo.GetProductionSheet(ctx, id); err != nil {
		return ErrProductionSheetNotFound
	}
	return s.repo.DeleteProductionSheet(ctx, id, strings.TrimSpace(byUsername))
}

func (s *Service) ListProductionSheets(ctx context.Context) ([]orderdomain.ProductionSheet, error) {
	sheets, err := s.repo.ListProductionSheets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list production sheets: %w", err)
	}
	return sheets, nil
}

func (s *Service) GetProductionSheet(ctx context.Context, id int64) (orderdomain.ProductionSheet, error) {
	sheet, err := s.repo.GetProductionSheet(ctx, id)
	if err != nil {
		return orderdomain.ProductionSheet{}, ErrProductionSheetNotFound
	}
	return sheet, nil
}

// validateProductionInput проверяет партию отработки: каждый заказ существует
// и не отменён, позиции принадлежат заказу, количества корректны. Возвращает
// ВСЕ заказы выбора (лист фиксирует партию); items — только отклонения от
// заявки, поэтому у заказа они могут быть пустыми.
func (s *Service) validateProductionInput(ctx context.Context, input RecordProductionInput) ([]OrderProductionInput, error) {
	if len(input.Orders) == 0 {
		return nil, apperr.Invalid("order.production_empty", "Нет заказов для отработки.")
	}
	result := make([]OrderProductionInput, 0, len(input.Orders))
	for _, orderInput := range input.Orders {
		number := strings.TrimSpace(orderInput.Number)
		if number == "" {
			return nil, apperr.Invalid("order.number_required", "Укажите номер заказа.")
		}
		existing, err := s.repo.GetOrderByNumber(ctx, number)
		if err != nil {
			return nil, ErrProductionOrderNotFound
		}
		if existing.Cancelled {
			return nil, apperr.Conflict("order.production_cancelled", fmt.Sprintf("Заказ %s отменён — отработка невозможна.", number))
		}
		byName := make(map[string]orderdomain.OrderItem, len(existing.Items))
		for _, item := range existing.Items {
			byName[strings.ToLower(strings.TrimSpace(item.ProductName))] = item
		}
		items := make([]ProducedItemInput, 0, len(orderInput.Items))
		seen := make(map[string]struct{}, len(orderInput.Items))
		for _, item := range orderInput.Items {
			name := strings.TrimSpace(item.ProductName)
			key := strings.ToLower(name)
			if name == "" {
				return nil, apperr.Invalid("order.production_item_name", "У позиции отработки должно быть название.")
			}
			if _, ok := seen[key]; ok {
				return nil, apperr.Invalid("order.production_item_duplicate", fmt.Sprintf("Позиция %q в отработке повторяется.", name))
			}
			seen[key] = struct{}{}
			orderItem, ok := byName[key]
			if !ok {
				return nil, apperr.Invalid("order.production_item_unknown", fmt.Sprintf("Позиции %q нет в заказе %s.", name, number))
			}
			if item.ProducedQuantity < 0 || math.IsNaN(item.ProducedQuantity) || math.IsInf(item.ProducedQuantity, 0) {
				return nil, apperr.Invalid("order.production_quantity", fmt.Sprintf("Укажите количество для позиции %q.", name))
			}
			loaded := orderItem.ProductionQuantity()
			if item.LoadedQuantity != nil {
				loaded = *item.LoadedQuantity
			}
			if loaded < 0 || math.IsNaN(loaded) || math.IsInf(loaded, 0) {
				return nil, apperr.Invalid("order.production_loaded_quantity", fmt.Sprintf("Укажите закладку для позиции %q.", name))
			}
			reason := strings.TrimSpace(item.Reason)
			if len([]rune(reason)) > 200 {
				return nil, apperr.Invalid("order.production_reason_too_long", fmt.Sprintf("Обоснование для %q слишком длинное (до 200 символов).", name))
			}
			items = append(items, ProducedItemInput{
				ProductName:      orderItem.ProductName,
				LoadedQuantity:   &loaded,
				ProducedQuantity: item.ProducedQuantity,
				IsDeviation:      item.ProducedQuantity != orderItem.ProductionQuantity(),
				Reason:           reason,
			})
		}
		result = append(result, OrderProductionInput{Number: number, Items: items})
	}
	return result, nil
}

// RestoreOrder clears an order's cancelled state. Active orders are returned
// unchanged (idempotent).
func (s *Service) RestoreOrder(ctx context.Context, number, byUsername string) (orderdomain.Order, error) {
	number = strings.TrimSpace(number)
	if number == "" {
		return orderdomain.Order{}, apperr.Invalid("order.number_required", "Укажите номер заказа.")
	}
	existing, err := s.repo.GetOrderByNumber(ctx, number)
	if err != nil {
		return orderdomain.Order{}, err
	}
	if !existing.Cancelled {
		return existing, nil
	}
	return s.repo.RestoreOrder(ctx, number)
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
			Code:       row.Code,
			Name:       row.Name,
			Theme:      row.Theme,
			CategoryID: row.CategoryID,
			SortOrder:  row.SortOrder,
		})
	}
	return items, nil
}

func (s *Service) ListOrderCategories(ctx context.Context) ([]orderdomain.OrderCategory, error) {
	categories, err := s.repo.ListOrderCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list order categories: %w", err)
	}
	return categories, nil
}

// CreateOrderCategory adds a new тип заявки. The letter goes into order
// numbers, the color must come from the fixed palette.
func (s *Service) CreateOrderCategory(ctx context.Context, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error) {
	category, err := sanitizeOrderCategoryInput(input)
	if err != nil {
		return orderdomain.OrderCategory{}, err
	}
	created, err := s.repo.CreateOrderCategory(ctx, category)
	if err != nil {
		return orderdomain.OrderCategory{}, fmt.Errorf("create order category: %w", err)
	}
	return created, nil
}

func (s *Service) UpdateOrderCategory(ctx context.Context, id int64, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error) {
	if id <= 0 {
		return orderdomain.OrderCategory{}, ErrCategoryNotFound
	}
	category, err := sanitizeOrderCategoryInput(input)
	if err != nil {
		return orderdomain.OrderCategory{}, err
	}
	updated, err := s.repo.UpdateOrderCategory(ctx, id, category)
	if err != nil {
		return orderdomain.OrderCategory{}, ErrCategoryNotFound
	}
	return updated, nil
}

// DeleteOrderCategory removes a category. Categories that still own dishes are
// protected — the admin reassigns dishes first. Existing orders keep working:
// their category link is severed (SET NULL), the number stays as issued.
func (s *Service) DeleteOrderCategory(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrCategoryNotFound
	}
	dishes, err := s.repo.CountDishesByCategoryID(ctx, id)
	if err != nil {
		return fmt.Errorf("count dishes by category: %w", err)
	}
	if dishes > 0 {
		return ErrCategoryHasDishes
	}
	return s.repo.DeleteOrderCategory(ctx, id)
}

// sanitizeOrderCategoryInput validates admin-supplied category fields. The code
// is derived from the name when empty; the letter is a single uppercase rune.
func sanitizeOrderCategoryInput(input orderdomain.OrderCategory) (orderdomain.OrderCategory, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return orderdomain.OrderCategory{}, apperr.Invalid("order.category_name_required", "Укажите название типа заявки.")
	}
	letter := []rune(strings.TrimSpace(input.Letter))
	if len(letter) == 0 {
		return orderdomain.OrderCategory{}, apperr.Invalid("order.category_letter_required", "Укажите букву для номера заказа.")
	}
	color := strings.TrimSpace(input.Color)
	if color == "" {
		color = "stone"
	}
	if !orderdomain.IsValidCategoryColor(color) {
		return orderdomain.OrderCategory{}, apperr.Invalid("order.category_color_invalid", "Недопустимый цвет типа заявки.")
	}
	code := strings.TrimSpace(input.Code)
	if code == "" {
		code = strings.ToLower(strings.Join(strings.Fields(name), "-"))
	}
	return orderdomain.OrderCategory{
		Code:         code,
		Letter:       strings.ToUpper(string(letter[0])),
		Name:         name,
		Color:        color,
		SortOrder:    max(input.SortOrder, 0),
		MonitorCodes: normalizeMonitorCodes(input.MonitorCodes),
	}, nil
}

// normalizeMonitorCodes trims, de-duplicates and drops empty dough codes,
// preserving the admin's order.
func normalizeMonitorCodes(codes []string) []string {
	result := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result
}

// SearchAvailableDishes returns iiko DISH products matching the query (by name
// or code) that an admin may add to the catalog. Results are capped so the UI
// never loads the whole product list.
func (s *Service) SearchAvailableDishes(ctx context.Context, query string, limit int) ([]orderdomain.AvailableDish, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	dishes, err := s.repo.SearchAvailableDishes(ctx, strings.TrimSpace(query), limit)
	if err != nil {
		return nil, fmt.Errorf("search available dishes: %w", err)
	}
	return dishes, nil
}

// AddDishCatalogItem inserts a dish into the catalog. The code must reference an
// existing iiko DISH product — admins pick only from dishes with tech cards.
func (s *Service) AddDishCatalogItem(ctx context.Context, input orderdomain.DishCatalogItem) (orderdomain.DishCatalogItem, error) {
	item, err := sanitizeDishCatalogInput(input)
	if err != nil {
		return orderdomain.DishCatalogItem{}, err
	}
	if err := s.requireExistingDish(ctx, item.Code); err != nil {
		return orderdomain.DishCatalogItem{}, err
	}
	if err := s.repo.UpsertDishCatalogItem(ctx, item); err != nil {
		return orderdomain.DishCatalogItem{}, fmt.Errorf("add dish catalog item: %w", err)
	}
	return toDomainDishCatalogItem(item), nil
}

// ReorderDishCatalog assigns sequential sort_order values (1..N) to the given
// codes, persisting a new drag-and-drop ordering.
func (s *Service) ReorderDishCatalog(ctx context.Context, codes []string) error {
	for i, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if err := s.repo.SetDishCatalogSortOrder(ctx, code, int64(i+1)); err != nil {
			return fmt.Errorf("reorder dish catalog: %w", err)
		}
	}
	return nil
}

// requireExistingDish rejects codes that are not present as an iiko DISH.
func (s *Service) requireExistingDish(ctx context.Context, code string) error {
	exists, err := s.repo.DishExistsByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("check dish exists: %w", err)
	}
	if !exists {
		return apperr.Invalid("order.dish_not_in_iiko", "Можно добавлять только блюда с техкартой из базы.")
	}
	return nil
}

// UpdateDishCatalogItem edits the dish identified by code. The code itself can
// be changed (the new code must stay unique).
func (s *Service) UpdateDishCatalogItem(ctx context.Context, code string, input orderdomain.DishCatalogItem) (orderdomain.DishCatalogItem, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return orderdomain.DishCatalogItem{}, apperr.Invalid("order.dish_code_required", "Укажите блюдо.")
	}
	item, err := sanitizeDishCatalogInput(input)
	if err != nil {
		return orderdomain.DishCatalogItem{}, err
	}
	updated, err := s.repo.UpdateDishCatalogItem(ctx, code, item)
	if err != nil {
		return orderdomain.DishCatalogItem{}, err
	}
	return toDomainDishCatalogItem(updated), nil
}

// DeleteDishCatalogItem removes a dish from the catalog by code.
func (s *Service) DeleteDishCatalogItem(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return apperr.Invalid("order.dish_code_required", "Укажите блюдо.")
	}
	return s.repo.DeleteDishCatalogItem(ctx, code)
}

// sanitizeDishCatalogInput validates and normalizes admin-supplied dish fields,
// deriving a code from the name when none is given.
func sanitizeDishCatalogInput(input orderdomain.DishCatalogItem) (DishCatalogItem, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return DishCatalogItem{}, apperr.Invalid("order.dish_name_required", "Укажите название блюда.")
	}
	code := strings.TrimSpace(input.Code)
	if code == "" {
		code = "custom:" + strings.ToLower(strings.Join(strings.Fields(name), " "))
	}
	return DishCatalogItem{
		Code:       code,
		Name:       name,
		Theme:      strings.TrimSpace(input.Theme),
		CategoryID: input.CategoryID,
		SortOrder:  max(input.SortOrder, 0),
	}, nil
}

func toDomainDishCatalogItem(item DishCatalogItem) orderdomain.DishCatalogItem {
	return orderdomain.DishCatalogItem{
		Code:       item.Code,
		Name:       item.Name,
		Theme:      item.Theme,
		CategoryID: item.CategoryID,
		SortOrder:  item.SortOrder,
	}
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

// EnsureDefaultOrderTemplates сидит каталог блюд из файлов-шаблонов, привязывая
// блюда к типам заявок. Upsert не затирает категорию, назначенную админом
// вручную; отсутствующий файл — не ошибка (шаблон опционален).
func (s *Service) EnsureDefaultOrderTemplates(ctx context.Context, seeds ...CatalogSeed) (EnsureDefaultTemplatesResult, error) {
	var result EnsureDefaultTemplatesResult
	if len(seeds) == 0 {
		seeds = []CatalogSeed{
			{Path: "templates/dishes.txt", CategoryCode: "buns"},
			{Path: "templates/bread.txt", CategoryCode: "bread"},
		}
	}

	categoryByCode := make(map[string]int64)
	if categories, err := s.repo.ListOrderCategories(ctx); err == nil {
		for _, category := range categories {
			categoryByCode[category.Code] = category.ID
		}
	} else {
		slog.WarnContext(ctx, "list categories for catalog seed failed", "error", err)
	}

	// Сквозная нумерация sort_order по всем файлам, чтобы группы не
	// перемешивались между типами.
	sortOffset := int64(0)
	for _, seed := range seeds {
		data, err := os.ReadFile(seed.Path) //nolint:gosec // path is a configured local template file.
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return result, fmt.Errorf("read default templates %s: %w", seed.Path, err)
		}

		var categoryID *int64
		if id, ok := categoryByCode[strings.TrimSpace(seed.CategoryCode)]; ok {
			categoryID = &id
		} else if seed.CategoryCode != "" {
			slog.WarnContext(ctx, "catalog seed category not found, seeding without category",
				"path", seed.Path, "category_code", seed.CategoryCode)
		}

		items := parseDefaultDishCatalogItems(string(data))
		for _, item := range items {
			item.CategoryID = categoryID
			item.SortOrder += sortOffset
			if err := s.repo.UpsertDishCatalogItem(ctx, item); err != nil {
				return result, fmt.Errorf("upsert dish catalog item %s: %w", item.Code, err)
			}
			result.CatalogItems++
		}
		sortOffset += int64(len(items))
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

func (s *Service) SaveOrderDraft(ctx context.Context, input SaveOrderDraftInput) (orderdomain.OrderDraft, error) {
	if s.repo == nil {
		return orderdomain.OrderDraft{}, fmt.Errorf("missing order repository")
	}
	validated, err := s.validateOrderWrite(ctx, input.Items, input.CategoryID, input.FromDepartmentID, input.FulfillmentDate, time.Now().UTC())
	if err != nil {
		return orderdomain.OrderDraft{}, err
	}
	draft, err := s.repo.SaveOrderDraft(ctx, SaveOrderDraftRepositoryInput{
		CreatedByUsername: input.CreatedByUsername,
		CategoryID:        validated.Category.ID,
		FromDepartmentID:  validated.Source.ID,
		Items:             validated.Items,
		FulfillmentDate:   validated.FulfillmentDate,
		Comments:          input.Comments,
	})
	if err != nil {
		return orderdomain.OrderDraft{}, err
	}
	return draft, nil
}

func (s *Service) GetOrderDraft(ctx context.Context, username string, categoryID int64) (orderdomain.OrderDraft, error) {
	if s.repo == nil {
		return orderdomain.OrderDraft{}, fmt.Errorf("missing order repository")
	}
	return s.repo.GetOrderDraft(ctx, strings.TrimSpace(username), categoryID)
}

func (s *Service) ListOrderDrafts(ctx context.Context, username string) ([]orderdomain.OrderDraft, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("missing order repository")
	}
	return s.repo.ListOrderDrafts(ctx, strings.TrimSpace(username))
}

func (s *Service) DeleteOrderDraft(ctx context.Context, username string, categoryID int64) error {
	if s.repo == nil {
		return fmt.Errorf("missing order repository")
	}
	return s.repo.DeleteOrderDraft(ctx, strings.TrimSpace(username), categoryID)
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
		} else {
			slog.InfoContext(ctx, "old orders cleanup finished", "deleted", deleted, "retention", retention.String())
		}
		cutoff := time.Now().UTC().Add(-retention)
		draftsDeleted, err := s.repo.DeleteOrderDraftsOlderThan(ctx, cutoff)
		if err != nil {
			slog.ErrorContext(ctx, "old order drafts cleanup failed", "error", err)
			return
		}
		slog.InfoContext(ctx, "old order drafts cleanup finished", "deleted", draftsDeleted, "retention", retention.String())
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

func (s *Service) orderSourceDepartment(ctx context.Context, departmentID *int64) (Department, error) {
	if departmentID == nil {
		return Department{}, fmt.Errorf("order source department is required")
	}
	department, err := s.repo.GetDepartmentByID(ctx, *departmentID)
	if err != nil {
		return Department{}, fmt.Errorf("get order source department: %w", err)
	}
	departmentType := enum.DepartmentType(strings.ToLower(strings.TrimSpace(department.Type)))
	if departmentType != enum.DepartmentTypeShop && departmentType != enum.DepartmentTypeWorkshop {
		return Department{}, fmt.Errorf("order source must be a shop or workshop department")
	}
	if strings.TrimSpace(department.Code) == "" && strings.TrimSpace(department.Name) == "" {
		return Department{}, fmt.Errorf("order source department code or name is required")
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
