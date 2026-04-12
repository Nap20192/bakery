package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"bakery/internal/app"
	"bakery/internal/domain/order"
)

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
		return
	}

	status := r.URL.Query().Get("status")
	var orders []*order.Order
	if status != "" {
		orders, err = s.order.ListByDateAndStatus(r.Context(), date, order.Status(status))
	} else {
		orders, err = s.order.ListByDate(r.Context(), date)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orderDTOs(orders))
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	claims, _ := ClaimsFromContext(r.Context())

	var body struct {
		ClientID          uuid.UUID  `json:"client_id"`
		KitchenID         *uuid.UUID `json:"kitchen_id"`
		TelegramChatID    int64      `json:"telegram_chat_id"`
		TelegramMessageID int64      `json:"telegram_message_id"`
		OrderDate         string     `json:"order_date"`
		RawText           string     `json:"raw_text"`
		Note              string     `json:"note"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := s.auth.CanCreateOrderAs(r.Context(), claims, body.ClientID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	orderDate, err := time.Parse("2006-01-02", body.OrderDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order_date, use YYYY-MM-DD")
		return
	}

	o, err := s.order.Create(r.Context(), app.CreateOrderInput{
		ClientID:          body.ClientID,
		KitchenID:         body.KitchenID,
		TelegramChatID:    body.TelegramChatID,
		TelegramMessageID: body.TelegramMessageID,
		OrderDate:         orderDate,
		RawText:           body.RawText,
		Note:              body.Note,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, orderDTO(o))
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	claims, _ := ClaimsFromContext(r.Context())

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	o, err := s.order.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := s.auth.CanViewOrder(r.Context(), claims, o); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orderDTO(o))
}

func (s *Server) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	claims, _ := ClaimsFromContext(r.Context())
	if err := s.auth.CanChangeOrderStatus(claims); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	o, err := s.order.UpdateStatus(r.Context(), id, order.Status(body.Status))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orderDTO(o))
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.order.Cancel(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) addOrderItem(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		ProductID uuid.UUID `json:"product_id"`
		Quantity  float64   `json:"quantity"`
		Unit      string    `json:"unit"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := s.order.AddItem(r.Context(), app.AddOrderItemInput{
		OrderID:   orderID,
		ProductID: body.ProductID,
		Quantity:  body.Quantity,
		Unit:      body.Unit,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, orderItemDTO(item))
}

func (s *Server) listOrderItems(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	items, err := s.order.ListItems(r.Context(), orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orderItemDTOs(items))
}
