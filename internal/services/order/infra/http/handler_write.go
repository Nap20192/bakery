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

	"bakery/internal/inbound/api/httpx"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
)

const (
	workshopDepartmentCode = "pekari"
	maxOrderItems          = 250
	maxItemQuantity        = 100000
)

type orderWriteRequest struct {
	Items           []orderWriteItem `json:"items"`
	FulfillmentDate string           `json:"fulfillment_date"`
}

type orderWriteItem struct {
	ProductName      string  `json:"product_name"`
	Quantity         float64 `json:"quantity"`
	ReservedQuantity float64 `json:"reserved_quantity"`
}

func (h *Handler) handleCatalog(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, _ := httpx.MiniAppUserFromContext(r.Context())
	if !httpx.IsShopUser(user) {
		httpx.WriteError(w, http.StatusForbidden, "Каталог нужен только магазину для создания заказа.")
		return
	}
	items, err := h.orderSvc.ListDishCatalog(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list dish catalog failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось загрузить список блюд.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil || h.departmentSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, ok := h.shopOrderWriter(w, r)
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
	fromDepartmentID := user.DepartmentID
	order, err := h.orderSvc.CreateOrder(r.Context(), orderdomain.CreateOrderInput{
		Items:             input.items,
		FromDepartmentID:  &fromDepartmentID,
		ToDepartmentID:    &toDepartmentID,
		CreatedByUsername: miniAppOrderAuthor(user),
		FulfillmentDate:   input.fulfillmentDate,
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
	user, ok := h.shopOrderWriter(w, r)
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

	fromDepartmentID := user.DepartmentID
	order, err := h.orderSvc.UpdateOrder(r.Context(), orderuc.UpdateOrderInput{
		Number:            number,
		Items:             input.items,
		FromDepartmentID:  &fromDepartmentID,
		ToDepartmentID:    existing.ToDepartmentID,
		CreatedByUsername: miniAppOrderAuthor(user),
		FulfillmentDate:   input.fulfillmentDate,
	})
	if err != nil {
		slog.WarnContext(r.Context(), "mini app update order failed", "error", err, "telegram_id", user.TelegramID)
		httpx.WriteAppError(w, r, err, "Не удалось обновить заказ. Проверьте позиции и количества.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.presenter.BuildOrderResponse(r.Context(), order))
}

type validatedOrderWrite struct {
	items           []orderdomain.OrderItem
	fulfillmentDate time.Time
}

func decodeOrderWriteRequest(w http.ResponseWriter, r *http.Request) (validatedOrderWrite, bool) {
	var request orderWriteRequest
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
	return validatedOrderWrite{items: items, fulfillmentDate: fulfillmentDate}, true
}

func validOrderQuantity(quantity float64) bool {
	return quantity >= 0 && quantity <= maxItemQuantity && !math.IsNaN(quantity) && !math.IsInf(quantity, 0) &&
		quantity == math.Trunc(quantity)
}

func (h *Handler) shopOrderWriter(w http.ResponseWriter, r *http.Request) (httpx.MiniAppUser, bool) {
	user, ok := httpx.MiniAppUserFromContext(r.Context())
	if !ok || !httpx.IsShopUser(user) {
		httpx.WriteError(w, http.StatusForbidden, "Только магазин может создавать и изменять заказы.")
		return httpx.MiniAppUser{}, false
	}
	return user, true
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
