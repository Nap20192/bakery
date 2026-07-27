package orderhttp

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"bakery/internal/inbound/api/contract"
	"bakery/internal/inbound/api/httpx"
	"bakery/internal/pkg/enum"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
)

func (h *Handler) handleSaveOrderDraft(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, ok := h.orderDraftWriter(w, r)
	if !ok {
		return
	}
	input, ok := decodeOrderWriteRequest(w, r)
	if !ok {
		return
	}
	draft, err := h.orderSvc.SaveOrderDraft(r.Context(), orderuc.SaveOrderDraftInput{
		CreatedByUsername: miniAppOrderAuthor(user),
		CategoryID:        input.categoryID,
		FromDepartmentID:  input.fromDepartmentID,
		Items:             input.items,
		FulfillmentDate:   input.fulfillmentDate,
		Comments:          input.comments,
	})
	if err != nil {
		slog.WarnContext(r.Context(), "mini app save order draft failed", "error", err, "telegram_id", user.TelegramID)
		httpx.WriteAppError(w, r, err, "Не удалось сохранить черновик. Проверьте позиции и количества.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, buildOrderDraftResponse(draft))
}

func (h *Handler) handleGetOrderDraft(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, ok := h.orderDraftWriter(w, r)
	if !ok {
		return
	}
	categoryID, ok := parseDraftCategoryID(w, r)
	if !ok {
		return
	}
	draft, err := h.orderSvc.GetOrderDraft(r.Context(), miniAppOrderAuthor(user), categoryID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Черновик не найден.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, buildOrderDraftResponse(draft))
}

func (h *Handler) handleListOrderDrafts(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, ok := h.orderDraftWriter(w, r)
	if !ok {
		return
	}
	drafts, err := h.orderSvc.ListOrderDrafts(r.Context(), miniAppOrderAuthor(user))
	if err != nil {
		slog.ErrorContext(r.Context(), "list order drafts failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось загрузить черновики.")
		return
	}
	responses := make([]contract.OrderDraft, 0, len(drafts))
	for _, draft := range drafts {
		responses = append(responses, buildOrderDraftResponse(draft))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) handleDeleteOrderDraft(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}
	user, ok := h.orderDraftWriter(w, r)
	if !ok {
		return
	}
	categoryID, ok := parseDraftCategoryID(w, r)
	if !ok {
		return
	}
	if err := h.orderSvc.DeleteOrderDraft(r.Context(), miniAppOrderAuthor(user), categoryID); err != nil {
		slog.WarnContext(r.Context(), "mini app delete order draft failed", "error", err, "telegram_id", user.TelegramID)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось удалить черновик.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// orderDraftWriter gates draft endpoints to the shop role — drafts are a
// shop-only convenience on the order-creation form.
func (h *Handler) orderDraftWriter(w http.ResponseWriter, r *http.Request) (httpx.MiniAppUser, bool) {
	user, ok := httpx.MiniAppUserFromContext(r.Context())
	if !ok || enum.NormalizeRole(user.Auth.Role) != enum.RoleShop {
		httpx.WriteError(w, http.StatusForbidden, "Черновики доступны только магазину.")
		return httpx.MiniAppUser{}, false
	}
	return user, true
}

func parseDraftCategoryID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	categoryID, err := strconv.ParseInt(r.PathValue("categoryId"), 10, 64)
	if err != nil || categoryID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректный тип заявки.")
		return 0, false
	}
	return categoryID, true
}

func buildOrderDraftResponse(draft orderdomain.OrderDraft) contract.OrderDraft {
	items := make([]contract.OrderItemWrite, 0, len(draft.Items))
	for _, item := range draft.Items {
		items = append(items, contract.OrderItemWrite{
			ProductName:      item.ProductName,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		})
	}
	fulfillmentDate := ""
	if !draft.FulfillmentDate.IsZero() {
		fulfillmentDate = draft.FulfillmentDate.Format("2006-01-02")
	}
	updatedAt := ""
	if !draft.UpdatedAt.IsZero() {
		updatedAt = draft.UpdatedAt.Format(time.RFC3339)
	}
	return contract.OrderDraft{
		Write: contract.OrderWrite{
			Items:            items,
			FulfillmentDate:  fulfillmentDate,
			FromDepartmentID: draft.FromDepartmentID,
			CategoryID:       draft.CategoryID,
			Comments:         buildCommentsResponse(draft.Comments),
		},
		UpdatedAt: updatedAt,
	}
}
