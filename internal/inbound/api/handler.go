package api

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	accessdomain "bakery/internal/domain/access"
	monitoringdomain "bakery/internal/domain/monitoring"
	orderdomain "bakery/internal/domain/order"
	"bakery/internal/pkg/enum"
	orderuc "bakery/internal/services/order/usecase/order"
)

var defaultMonitorCodes = []string{"17642", "17644", "17650", "19694"}

type departmentResponse struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type meResponse struct {
	Role           string `json:"role"`
	TelegramID     int64  `json:"telegram_id"`
	TelegramUser   string `json:"telegram_username"`
	DepartmentID   int64  `json:"department_id"`
	DepartmentCode string `json:"department_code"`
	DepartmentName string `json:"department_name"`
	DepartmentType string `json:"department_type"`
}

type orderItemResponse struct {
	Code               string  `json:"code"`
	ProductName        string  `json:"product_name"`
	Quantity           float64 `json:"quantity"`
	ReservedQuantity   float64 `json:"reserved_quantity"`
	ProductionQuantity float64 `json:"production_quantity"`
}

type orderHistoryItemResponse struct {
	ChangeType          string   `json:"change_type"`
	ProductCode         string   `json:"product_code"`
	ProductName         string   `json:"product_name"`
	OldQuantity         *float64 `json:"old_quantity,omitempty"`
	NewQuantity         *float64 `json:"new_quantity,omitempty"`
	OldReservedQuantity *float64 `json:"old_reserved_quantity,omitempty"`
	NewReservedQuantity *float64 `json:"new_reserved_quantity,omitempty"`
}

type orderHistoryResponse struct {
	ID                int64                      `json:"id"`
	ChangedByUsername string                     `json:"changed_by_username"`
	ChangedAt         string                     `json:"changed_at"`
	Items             []orderHistoryItemResponse `json:"items"`
}

type orderResponse struct {
	ID                string                 `json:"id"`
	Number            string                 `json:"number"`
	Location          string                 `json:"location"`
	CreatedByUsername string                 `json:"created_by_username"`
	FromDepartment    *departmentResponse    `json:"from_department,omitempty"`
	ToDepartment      *departmentResponse    `json:"to_department,omitempty"`
	Items             []orderItemResponse    `json:"items"`
	CreatedAt         string                 `json:"created_at"`
	FulfillmentDate   string                 `json:"fulfillment_date"`
	MonitorCommand    string                 `json:"monitor_command"`
	History           []orderHistoryResponse `json:"history,omitempty"`
}

type ordersPageResponse struct {
	Items      []orderResponse `json:"items"`
	Page       int32           `json:"page"`
	Limit      int32           `json:"limit"`
	Offset     int32           `json:"offset"`
	Total      int64           `json:"total"`
	TotalPages int32           `json:"total_pages"`
}

type monitorReportResponse struct {
	Code   string                            `json:"code"`
	Report monitoringdomain.IngredientReport `json:"report"`
}
type monitorResponse struct {
	Reports []monitorReportResponse `json:"reports"`
	Order   orderResponse           `json:"order"`
}

type batchMonitorOrderResponse struct {
	Order   orderResponse           `json:"order"`
	Reports []monitorReportResponse `json:"reports"`
}

type batchMonitorResponse struct {
	Orders       []batchMonitorOrderResponse `json:"orders"`
	TotalReports []monitorReportResponse     `json:"total_reports"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	viewer, _ := miniAppUserFromContext(r.Context())
	resp := meResponse{
		Role:         viewer.Auth.Role,
		TelegramUser: strings.TrimSpace(telegramUsernameOf(viewer.Auth)),
	}
	if viewer.Auth.TelegramID != nil {
		resp.TelegramID = *viewer.Auth.TelegramID
	}
	// Department is optional (admins may have none). Resolve it when present.
	if viewer.Auth.DepartmentID != nil {
		if department, err := s.departmentSvc.GetByID(r.Context(), *viewer.Auth.DepartmentID); err == nil {
			resp.DepartmentID = department.ID
			resp.DepartmentCode = department.Code
			resp.DepartmentName = department.Name
			resp.DepartmentType = department.Type
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func telegramUsernameOf(u accessdomain.AuthUser) string {
	if u.TelegramUsername != nil {
		return *u.TelegramUsername
	}
	return ""
}

func (s *Server) handleListDepartments(w http.ResponseWriter, r *http.Request) {
	if s.departmentSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис магазинов временно недоступен."})
		return
	}
	departmentType := enum.DepartmentType(trim(r.URL.Query().Get("type")))
	user, _ := miniAppUserFromContext(r.Context())
	if isShopMiniAppUser(user) {
		if departmentType != "" && departmentType != enum.DepartmentTypeShop {
			writeJSON(w, http.StatusOK, []departmentResponse{})
			return
		}
		writeJSON(w, http.StatusOK, []departmentResponse{{
			ID:   user.DepartmentID,
			Code: user.DepartmentCode,
			Name: user.DepartmentName,
			Type: user.DepartmentType,
		}})
		return
	}

	departments, err := s.departmentSvc.ListByType(r.Context(), departmentType)
	if err != nil {
		slog.ErrorContext(r.Context(), "list departments failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось получить список магазинов."})
		return
	}
	response := make([]departmentResponse, 0, len(departments))
	for _, department := range departments {
		response = append(response, departmentResponse{
			ID:   department.ID,
			Code: department.Code,
			Name: department.Name,
			Type: department.Type,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	if s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис заказов временно недоступен."})
		return
	}

	user, _ := miniAppUserFromContext(r.Context())
	limit := int32(10)
	if raw := trim(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Параметр limit должен быть положительным числом."})
			return
		}
		limit = int32(parsed)
	}
	page := int32(1)
	if raw := trim(r.URL.Query().Get("page")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Параметр page должен быть положительным числом."})
			return
		}
		page = int32(parsed)
	}
	offset := (page - 1) * limit
	var fromDepartmentID *int64
	if isShopMiniAppUser(user) {
		fromDepartmentID = &user.DepartmentID
	} else if raw := trim(r.URL.Query().Get("from_department_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Параметр from_department_id должен быть положительным числом."})
			return
		}
		fromDepartmentID = &parsed
	}
	fulfillmentDate, err := parseRequestDate(r.URL.Query().Get("fulfillment_date"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Дата выполнения должна быть в формате YYYY-MM-DD."})
		return
	}

	result, err := s.orderSvc.ListOrders(r.Context(), orderuc.ListOrdersInput{
		Limit:            limit,
		Offset:           offset,
		FromDepartmentID: fromDepartmentID,
		FulfillmentDate:  fulfillmentDate,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list orders failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось получить список заказов."})
		return
	}

	writeJSON(w, http.StatusOK, ordersPageResponse{
		Items:      s.buildOrderResponses(r.Context(), result.Orders),
		Page:       page,
		Limit:      result.Limit,
		Offset:     result.Offset,
		Total:      result.Total,
		TotalPages: int32(math.Ceil(float64(result.Total) / float64(result.Limit))),
	})
}

func (s *Server) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	if s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис заказов временно недоступен."})
		return
	}

	id := trim(r.PathValue("id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Укажите номер заказа."})
		return
	}

	order, err := s.orderSvc.GetOrderByNumber(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", id)})
		return
	}
	if !miniAppUserCanReadOrder(r.Context(), order) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", id)})
		return
	}

	writeJSON(w, http.StatusOK, s.buildOrderResponse(r.Context(), order))
}

func (s *Server) handleMonitorDefault(w http.ResponseWriter, r *http.Request) {
	if s.monitorSvc == nil || s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис калькуляции временно недоступен."})
		return
	}

	orderID := trim(r.PathValue("id"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Укажите номер заказа."})
		return
	}

	order, err := s.orderSvc.GetOrderByNumber(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", orderID)})
		return
	}
	if !miniAppUserCanReadOrder(r.Context(), order) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", orderID)})
		return
	}

	reports := make([]monitorReportResponse, 0, len(defaultMonitorCodes))
	for _, code := range defaultMonitorCodes {
		report, err := s.monitorSvc.GetIngredientsByCode(r.Context(), code, order)
		if err != nil {
			slog.WarnContext(r.Context(), "default monitor calculation failed", "order_number", orderID, "code", code, "error", err)
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Не удалось посчитать калькуляцию. Проверьте заказ и техкарты."})
			return
		}
		reports = append(reports, monitorReportResponse{Code: code, Report: report})
	}

	writeJSON(w, http.StatusOK, s.buildMonitorResponse(r.Context(), order, reports))
}

func (s *Server) handleMonitorByProduct(w http.ResponseWriter, r *http.Request) {
	if s.monitorSvc == nil || s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис калькуляции временно недоступен."})
		return
	}

	orderID := trim(r.PathValue("id"))
	productCode := trim(r.PathValue("product_id"))
	if orderID == "" || productCode == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Укажите номер заказа и код продукта."})
		return
	}

	order, err := s.orderSvc.GetOrderByNumber(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", orderID)})
		return
	}
	if !miniAppUserCanReadOrder(r.Context(), order) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", orderID)})
		return
	}

	report, err := s.monitorSvc.GetIngredientsByCode(r.Context(), productCode, order)
	if err != nil {
		slog.WarnContext(r.Context(), "monitor calculation failed", "order_number", orderID, "code", productCode, "error", err)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Не удалось посчитать калькуляцию по этому продукту. Проверьте код продукта и техкарту."})
		return
	}

	writeJSON(w, http.StatusOK, s.buildMonitorResponse(r.Context(), order, []monitorReportResponse{
		{Code: productCode, Report: report},
	}))
}

func (s *Server) handleMonitorBatch(w http.ResponseWriter, r *http.Request) {
	if s.monitorSvc == nil || s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис калькуляции временно недоступен."})
		return
	}

	orderNumbers := uniqueQueryValues(r.URL.Query()["orders"])
	if len(orderNumbers) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Укажите хотя бы один заказ в параметре orders."})
		return
	}
	if len(orderNumbers) > 50 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Слишком много заказов. Выберите не больше 50."})
		return
	}

	orders := make([]orderdomain.Order, 0, len(orderNumbers))
	for _, number := range orderNumbers {
		order, err := s.orderSvc.GetOrderByNumber(r.Context(), number)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", number)})
			return
		}
		if !miniAppUserCanReadOrder(r.Context(), order) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", number)})
			return
		}
		orders = append(orders, order)
	}

	report, err := s.monitorSvc.GetBatchIngredientsByCodes(r.Context(), defaultMonitorCodes, orders)
	if err != nil {
		slog.WarnContext(r.Context(), "batch monitor calculation failed", "orders", orderNumbers, "error", err)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Не удалось посчитать калькуляцию по выбранным заказам. Проверьте заказы и техкарты."})
		return
	}

	writeJSON(w, http.StatusOK, s.buildBatchMonitorResponse(r.Context(), orders, report))
}

func isShopMiniAppUser(user miniAppUser) bool {
	return strings.EqualFold(strings.TrimSpace(user.DepartmentType), string(enum.DepartmentTypeShop))
}

func isWorkshopMiniAppUser(user miniAppUser) bool {
	return strings.EqualFold(strings.TrimSpace(user.DepartmentType), string(enum.DepartmentTypeWorkshop))
}

func miniAppUserCanReadOrder(ctx context.Context, order orderdomain.Order) bool {
	user, ok := miniAppUserFromContext(ctx)
	if !ok {
		return false
	}
	if !isShopMiniAppUser(user) {
		return isWorkshopMiniAppUser(user)
	}
	return order.FromDepartmentID != nil && *order.FromDepartmentID == user.DepartmentID
}

func (s *Server) buildMonitorResponse(ctx context.Context, order orderdomain.Order, reports []monitorReportResponse) monitorResponse {
	return monitorResponse{
		Reports: reports,
		Order:   s.buildOrderResponse(ctx, order),
	}
}

func (s *Server) buildBatchMonitorResponse(
	ctx context.Context,
	orders []orderdomain.Order,
	report monitoringdomain.BatchMonitoringReport,
) batchMonitorResponse {
	orderByNumber := make(map[string]orderdomain.Order, len(orders))
	for _, order := range orders {
		orderByNumber[order.Number] = order
	}

	response := batchMonitorResponse{
		Orders:       make([]batchMonitorOrderResponse, 0, len(report.Orders)),
		TotalReports: make([]monitorReportResponse, 0, len(report.TotalReports)),
	}
	for i, total := range report.TotalReports {
		code := ""
		if i < len(defaultMonitorCodes) {
			code = defaultMonitorCodes[i]
		}
		response.TotalReports = append(response.TotalReports, monitorReportResponse{
			Code:   code,
			Report: total,
		})
	}
	for _, orderReport := range report.Orders {
		order := orderByNumber[orderReport.OrderNumber]
		reports := make([]monitorReportResponse, 0, len(orderReport.Reports))
		for i, item := range orderReport.Reports {
			code := ""
			if i < len(defaultMonitorCodes) {
				code = defaultMonitorCodes[i]
			}
			reports = append(reports, monitorReportResponse{
				Code:   code,
				Report: item,
			})
		}
		response.Orders = append(response.Orders, batchMonitorOrderResponse{
			Order:   s.buildOrderResponse(ctx, order),
			Reports: reports,
		})
	}
	return response
}

func (s *Server) buildOrderResponses(ctx context.Context, orders []orderdomain.Order) []orderResponse {
	responses := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, s.buildOrderResponse(ctx, order))
	}
	return responses
}

func (s *Server) buildOrderResponse(ctx context.Context, order orderdomain.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, orderItemResponse{
			Code:               item.Code,
			ProductName:        item.ProductName,
			Quantity:           item.Quantity,
			ReservedQuantity:   item.ReservedQuantity,
			ProductionQuantity: item.ProductionQuantity(),
		})
	}

	createdAt := ""
	if !order.CreatedAt.IsZero() {
		createdAt = order.CreatedAt.Format(time.RFC3339)
	}
	fulfillmentDate := ""
	if !order.FulfillmentDate.IsZero() {
		fulfillmentDate = order.FulfillmentDate.Format("2006-01-02")
	}

	return orderResponse{
		ID:                order.ID,
		Number:            order.Number,
		Location:          order.Location,
		CreatedByUsername: order.CreatedByUsername,
		FromDepartment:    s.departmentResponse(ctx, order.FromDepartmentID),
		ToDepartment:      s.departmentResponse(ctx, order.ToDepartmentID),
		Items:             items,
		CreatedAt:         createdAt,
		FulfillmentDate:   fulfillmentDate,
		MonitorCommand:    fmt.Sprintf("/monitor %s", order.Number),
		History:           buildOrderHistoryResponse(order.History),
	}
}

func buildOrderHistoryResponse(history []orderdomain.OrderHistory) []orderHistoryResponse {
	result := make([]orderHistoryResponse, 0, len(history))
	for _, row := range history {
		items := make([]orderHistoryItemResponse, 0, len(row.Items))
		for _, item := range row.Items {
			items = append(items, orderHistoryItemResponse{
				ChangeType:          item.ChangeType,
				ProductCode:         item.ProductCode,
				ProductName:         item.ProductName,
				OldQuantity:         item.OldQuantity,
				NewQuantity:         item.NewQuantity,
				OldReservedQuantity: item.OldReservedQuantity,
				NewReservedQuantity: item.NewReservedQuantity,
			})
		}
		changedAt := ""
		if !row.ChangedAt.IsZero() {
			changedAt = row.ChangedAt.Format(time.RFC3339)
		}
		result = append(result, orderHistoryResponse{
			ID:                row.ID,
			ChangedByUsername: row.ChangedByUsername,
			ChangedAt:         changedAt,
			Items:             items,
		})
	}
	return result
}

func (s *Server) departmentResponse(ctx context.Context, id *int64) *departmentResponse {
	if id == nil || s.departmentSvc == nil {
		return nil
	}
	department, err := s.departmentSvc.GetByID(ctx, *id)
	if err != nil {
		return &departmentResponse{ID: *id, Name: strconv.FormatInt(*id, 10)}
	}
	return &departmentResponse{
		ID:   department.ID,
		Code: department.Code,
		Name: department.Name,
		Type: department.Type,
	}
}

func uniqueQueryValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = trim(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
