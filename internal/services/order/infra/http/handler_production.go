package orderhttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"bakery/internal/inbound/api/httpx"
	authdomain "bakery/internal/services/auth/domain"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
)

// Журнал отработок (факт выпечки): каждая отработка — отдельный документ,
// который можно открыть, изменить и удалить. Заявки магазинов при этом не
// изменяются — факт проецируется на produced_quantity позиций.

type productionRequest struct {
	Orders []productionOrderRequest `json:"orders"`
}

type productionOrderRequest struct {
	Number string                  `json:"number"`
	Items  []productionItemRequest `json:"items"`
}

type productionItemRequest struct {
	ProductName      string  `json:"product_name"`
	ProducedQuantity float64 `json:"produced_quantity"`
	Reason           string  `json:"reason"`
}

type productionSheetResponse struct {
	ID                int64                         `json:"id"`
	CreatedByUsername string                        `json:"created_by_username"`
	CreatedAt         string                        `json:"created_at"`
	UpdatedAt         string                        `json:"updated_at"`
	OrderNumbers      []string                      `json:"order_numbers"`
	ItemCount         int64                         `json:"item_count"`
	Items             []productionSheetItemResponse `json:"items,omitempty"`
}

type productionSheetItemResponse struct {
	OrderNumber      string  `json:"order_number"`
	ProductName      string  `json:"product_name"`
	ProducedQuantity float64 `json:"produced_quantity"`
	Reason           string  `json:"reason"`
}

func toProductionSheetResponse(sheet orderdomain.ProductionSheet) productionSheetResponse {
	items := make([]productionSheetItemResponse, 0, len(sheet.Items))
	for _, item := range sheet.Items {
		items = append(items, productionSheetItemResponse{
			OrderNumber:      item.OrderNumber,
			ProductName:      item.ProductName,
			ProducedQuantity: item.ProducedQuantity,
			Reason:           item.Reason,
		})
	}
	orderNumbers := sheet.OrderNumbers
	if orderNumbers == nil {
		orderNumbers = []string{}
	}
	return productionSheetResponse{
		ID:                sheet.ID,
		CreatedByUsername: sheet.CreatedByUsername,
		CreatedAt:         sheet.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         sheet.UpdatedAt.Format(time.RFC3339),
		OrderNumbers:      orderNumbers,
		ItemCount:         sheet.ItemCount,
		Items:             items,
	}
}

// productionWriter пускает только пекаря и админа: отработку ведёт цех.
func (h *Handler) productionWriter(w http.ResponseWriter, r *http.Request) (httpx.MiniAppUser, bool) {
	user, ok := httpx.MiniAppUserFromContext(r.Context())
	role := authdomain.NormalizeRole(user.Auth.Role)
	if !ok || (role != authdomain.RoleBaker && role != authdomain.RoleAdmin) {
		httpx.WriteError(w, http.StatusForbidden, "Отработку ведёт пекарь или администратор.")
		return httpx.MiniAppUser{}, false
	}
	return user, true
}

func (h *Handler) decodeProductionRequest(w http.ResponseWriter, r *http.Request) (orderuc.RecordProductionInput, bool) {
	var request productionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Проверьте формат отработки.")
		return orderuc.RecordProductionInput{}, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		httpx.WriteError(w, http.StatusBadRequest, "Проверьте формат отработки.")
		return orderuc.RecordProductionInput{}, false
	}
	if len(request.Orders) == 0 || len(request.Orders) > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "Добавьте от 1 до 100 заказов в отработку.")
		return orderuc.RecordProductionInput{}, false
	}
	input := orderuc.RecordProductionInput{}
	for _, order := range request.Orders {
		items := make([]orderuc.ProducedItemInput, 0, len(order.Items))
		for _, item := range order.Items {
			items = append(items, orderuc.ProducedItemInput{
				ProductName:      item.ProductName,
				ProducedQuantity: item.ProducedQuantity,
				Reason:           item.Reason,
			})
		}
		input.Orders = append(input.Orders, orderuc.OrderProductionInput{Number: order.Number, Items: items})
	}
	return input, true
}

func (h *Handler) handleCreateProductionSheet(w http.ResponseWriter, r *http.Request) {
	user, ok := h.productionWriter(w, r)
	if !ok {
		return
	}
	input, ok := h.decodeProductionRequest(w, r)
	if !ok {
		return
	}
	input.ProducedByUsername = miniAppOrderAuthor(user)
	sheet, err := h.orderSvc.CreateProductionSheet(r.Context(), input)
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось сохранить отработку.")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProductionSheetResponse(sheet))
}

func (h *Handler) handleListProductionSheets(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.productionWriter(w, r); !ok {
		return
	}
	sheets, err := h.orderSvc.ListProductionSheets(r.Context())
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось загрузить журнал отработок.")
		return
	}
	responses := make([]productionSheetResponse, 0, len(sheets))
	for _, sheet := range sheets {
		responses = append(responses, toProductionSheetResponse(sheet))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) handleGetProductionSheet(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.productionWriter(w, r); !ok {
		return
	}
	id, ok := productionSheetID(w, r)
	if !ok {
		return
	}
	sheet, err := h.orderSvc.GetProductionSheet(r.Context(), id)
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось загрузить отработку.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProductionSheetResponse(sheet))
}

func (h *Handler) handleUpdateProductionSheet(w http.ResponseWriter, r *http.Request) {
	user, ok := h.productionWriter(w, r)
	if !ok {
		return
	}
	id, ok := productionSheetID(w, r)
	if !ok {
		return
	}
	input, ok := h.decodeProductionRequest(w, r)
	if !ok {
		return
	}
	input.ProducedByUsername = miniAppOrderAuthor(user)
	sheet, err := h.orderSvc.UpdateProductionSheet(r.Context(), id, input)
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось обновить отработку.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProductionSheetResponse(sheet))
}

func (h *Handler) handleDeleteProductionSheet(w http.ResponseWriter, r *http.Request) {
	user, ok := h.productionWriter(w, r)
	if !ok {
		return
	}
	id, ok := productionSheetID(w, r)
	if !ok {
		return
	}
	if err := h.orderSvc.DeleteProductionSheet(r.Context(), id, miniAppOrderAuthor(user)); err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось удалить отработку.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func productionSheetID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(httpx.Trim(r.PathValue("id")), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите отработку.")
		return 0, false
	}
	return id, true
}
