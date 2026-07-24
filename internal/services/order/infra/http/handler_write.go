package orderhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bakery/internal/inbound/api/contract"
	"bakery/internal/pkg/enum"

	"bakery/internal/inbound/api/httpx"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
)

const (
	workshopDepartmentCode = "pekari"
	maxOrderItems          = 250
	maxItemQuantity        = 100000
)

// handleCatalog returns the dish catalog. Shops build orders from it; bakers
// use it to group order positions by catalog theme in the order views.
func (h *Handler) handleCatalog(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	items, err := h.orderSvc.ListDishCatalog(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list dish catalog failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось загрузить список блюд.")
		return
	}
	responses := make([]contract.Dish, 0, len(items))
	for _, item := range items {
		responses = append(responses, toDishResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil || h.departmentSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, ok := h.orderWriter(w, r)
	if !ok {
		return
	}
	input, ok := decodeOrderWriteRequest(w, r)
	if !ok {
		return
	}

	toDepartmentID, ok := h.workshopDepartmentID(w, r)
	if !ok {
		return
	}
	var fromDepartmentID int64
	if enum.NormalizeRole(user.Auth.Role) == enum.RoleBaker {
		fromDepartmentID = toDepartmentID
	} else {
		fromDepartmentID, ok = h.resolveOrderSourceDepartmentID(w, r, input.fromDepartmentID, false)
	}
	if !ok {
		return
	}
	order, err := h.orderSvc.CreateOrder(r.Context(), orderdomain.CreateOrderInput{
		Items:             input.items,
		FromDepartmentID:  &fromDepartmentID,
		ToDepartmentID:    &toDepartmentID,
		CategoryID:        input.categoryID,
		CreatedByUsername: miniAppOrderAuthor(user),
		FulfillmentDate:   input.fulfillmentDate,
		Comments:          input.comments,
	})
	if err != nil {
		slog.WarnContext(r.Context(), "mini app create order failed", "error", err, "telegram_id", user.TelegramID)
		httpx.WriteAppError(w, r, err, "Не удалось создать заказ. Проверьте позиции и количества.")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, h.presenter.BuildOrderResponse(r.Context(), order))
}

func (h *Handler) handleUpdateOrder(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, ok := h.orderWriter(w, r)
	if !ok {
		return
	}
	number := httpx.Trim(r.PathValue("id"))
	if number == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите номер заказа.")
		return
	}
	existing, err := h.orderSvc.GetOrderByNumber(r.Context(), number)
	if err != nil || !UserCanReadOrder(r.Context(), existing) {
		httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", number))
		return
	}
	input, ok := decodeOrderWriteRequest(w, r)
	if !ok {
		return
	}

	fromDepartmentID := existing.FromDepartmentID
	if input.fromDepartmentID != nil {
		role := enum.NormalizeRole(user.Auth.Role)
		resolved, ok := h.resolveOrderSourceDepartmentID(
			w,
			r,
			input.fromDepartmentID,
			role == enum.RoleBaker || role == enum.RoleAdmin,
		)
		if !ok {
			return
		}
		fromDepartmentID = &resolved
	}
	order, err := h.orderSvc.UpdateOrder(r.Context(), orderuc.UpdateOrderInput{
		Number:            number,
		Items:             input.items,
		FromDepartmentID:  fromDepartmentID,
		ToDepartmentID:    existing.ToDepartmentID,
		CreatedByUsername: miniAppOrderAuthor(user),
		FulfillmentDate:   input.fulfillmentDate,
		Comments:          input.comments,
	})
	if err != nil {
		slog.WarnContext(r.Context(), "mini app update order failed", "error", err, "telegram_id", user.TelegramID)
		httpx.WriteAppError(w, r, err, "Не удалось обновить заказ. Проверьте позиции и количества.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.presenter.BuildOrderResponse(r.Context(), order))
}

func (h *Handler) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	h.changeOrderCancellation(w, r, true)
}

func (h *Handler) handleRestoreOrder(w http.ResponseWriter, r *http.Request) {
	h.changeOrderCancellation(w, r, false)
}

// changeOrderCancellation cancels or restores an order. Both are order-writer
// actions guarded by orderWriter; the actor is recorded on cancel.
func (h *Handler) changeOrderCancellation(w http.ResponseWriter, r *http.Request, cancel bool) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, ok := h.orderWriter(w, r)
	if !ok {
		return
	}
	number := httpx.Trim(r.PathValue("id"))
	if number == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите номер заказа.")
		return
	}
	existing, err := h.orderSvc.GetOrderByNumber(r.Context(), number)
	if err != nil || !UserCanReadOrder(r.Context(), existing) {
		httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", number))
		return
	}

	actor := miniAppOrderAuthor(user)
	var order orderdomain.Order
	if cancel {
		order, err = h.orderSvc.CancelOrder(r.Context(), number, actor)
	} else {
		order, err = h.orderSvc.RestoreOrder(r.Context(), number, actor)
	}
	if err != nil {
		slog.WarnContext(r.Context(), "mini app cancel/restore order failed", "error", err, "cancel", cancel, "telegram_id", user.TelegramID)
		httpx.WriteAppError(w, r, err, "Не удалось обновить статус заказа.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.presenter.BuildOrderResponse(r.Context(), order))
}

type validatedOrderWrite struct {
	items            []orderdomain.OrderItem
	fulfillmentDate  time.Time
	fromDepartmentID *int64
	categoryID       int64
	comments         orderdomain.OrderComments
}

func buildComments(req contract.Comments) orderdomain.OrderComments {
	out := orderdomain.OrderComments{General: strings.TrimSpace(req.General)}
	for _, c := range req.Items {
		name := strings.TrimSpace(c.ProductName)
		comment := strings.TrimSpace(c.Comment)
		if name == "" || comment == "" {
			continue
		}
		out.Items = append(out.Items, orderdomain.ItemComment{ProductName: name, Comment: comment})
	}
	return out
}

func decodeOrderWriteRequest(w http.ResponseWriter, r *http.Request) (validatedOrderWrite, bool) {
	var request contract.OrderWrite
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Проверьте формат заказа.")
		return validatedOrderWrite{}, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		httpx.WriteError(w, http.StatusBadRequest, "Проверьте формат заказа.")
		return validatedOrderWrite{}, false
	}
	if len(request.Items) == 0 || len(request.Items) > maxOrderItems {
		httpx.WriteError(w, http.StatusBadRequest, "Добавьте от 1 до 250 позиций заказа.")
		return validatedOrderWrite{}, false
	}

	fulfillmentDate, err := httpx.ParseRequestDate(request.FulfillmentDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Дата выполнения должна быть в формате YYYY-MM-DD.")
		return validatedOrderWrite{}, false
	}
	items := make([]orderdomain.OrderItem, 0, len(request.Items))
	seen := make(map[string]struct{}, len(request.Items))
	for _, item := range request.Items {
		name := strings.TrimSpace(item.ProductName)
		key := strings.ToLower(name)
		if name == "" {
			httpx.WriteError(w, http.StatusBadRequest, "У каждой позиции должно быть название блюда.")
			return validatedOrderWrite{}, false
		}
		if _, ok := seen[key]; ok {
			httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Позиция %q добавлена повторно.", name))
			return validatedOrderWrite{}, false
		}
		seen[key] = struct{}{}
		if !validOrderQuantity(item.Quantity) || !validOrderQuantity(item.ReservedQuantity) ||
			item.Quantity+item.ReservedQuantity <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Укажите целое количество для позиции %q.", name))
			return validatedOrderWrite{}, false
		}
		items = append(items, orderdomain.OrderItem{
			ProductName:      name,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		})
	}
	return validatedOrderWrite{
		items:            items,
		fulfillmentDate:  fulfillmentDate,
		fromDepartmentID: request.FromDepartmentID,
		categoryID:       request.CategoryID,
		comments:         buildComments(request.Comments),
	}, true
}

func validOrderQuantity(quantity float64) bool {
	return quantity >= 0 && quantity <= maxItemQuantity && !math.IsNaN(quantity) && !math.IsInf(quantity, 0) &&
		quantity == math.Trunc(quantity)
}

func (h *Handler) orderWriter(w http.ResponseWriter, r *http.Request) (httpx.MiniAppUser, bool) {
	user, ok := httpx.MiniAppUserFromContext(r.Context())
	if !ok || !httpx.CanWriteOrders(user) {
		httpx.WriteError(w, http.StatusForbidden, "Недостаточно прав для изменения заказов.")
		return httpx.MiniAppUser{}, false
	}
	return user, true
}

// resolveOrderSourceDepartmentID validates the department the order is sent
// from. Shop orders keep their existing picker; baker/admin edits may also
// preserve or select the workshop source.
func (h *Handler) resolveOrderSourceDepartmentID(
	w http.ResponseWriter,
	r *http.Request,
	id *int64,
	allowWorkshop bool,
) (int64, bool) {
	if id == nil || *id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Выберите отправителя заказа.")
		return 0, false
	}
	department, err := h.departmentSvc.GetByID(r.Context(), *id)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Выбранный отправитель не найден.")
		return 0, false
	}
	departmentType := enum.DepartmentType(strings.ToLower(strings.TrimSpace(department.Type)))
	if departmentType != enum.DepartmentTypeShop &&
		(!allowWorkshop || departmentType != enum.DepartmentTypeWorkshop) {
		httpx.WriteError(w, http.StatusBadRequest, "Недопустимый отправитель заказа.")
		return 0, false
	}
	return department.ID, true
}

func (h *Handler) workshopDepartmentID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	department, err := h.departmentSvc.GetByCode(r.Context(), workshopDepartmentCode)
	if err != nil {
		slog.ErrorContext(r.Context(), "workshop department lookup failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Цех не настроен. Не удалось создать заказ.")
		return 0, false
	}
	return department.ID, true
}

func miniAppOrderAuthor(user httpx.MiniAppUser) string {
	if name := strings.TrimSpace(user.TelegramUser); name != "" {
		return name
	}
	if user.Auth.TelegramUsername != nil && strings.TrimSpace(*user.Auth.TelegramUsername) != "" {
		return strings.TrimSpace(*user.Auth.TelegramUsername)
	}
	if name := strings.TrimSpace(user.Auth.Username); name != "" {
		return name
	}
	return "telegram_" + strconv.FormatInt(user.TelegramID, 10)
}
