// Package departmenthttp is the HTTP delivery adapter of the department service.
package departmenthttp

import (
	"log/slog"
	"net/http"

	"bakery/internal/inbound/api/httpx"
	"bakery/internal/pkg/enum"
	departmentuc "bakery/internal/services/department/usecase/department"
)

// Handler is the department service HTTP delivery adapter.
type Handler struct {
	departmentSvc departmentuc.UseCase
}

func New(departmentSvc departmentuc.UseCase) *Handler {
	return &Handler{departmentSvc: departmentSvc}
}

// RegisterRoutes wires the department endpoints behind Mini App authentication.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("GET /departments", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleListDepartments)))
}

func (h *Handler) handleListDepartments(w http.ResponseWriter, r *http.Request) {
	if h.departmentSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис магазинов временно недоступен.")
		return
	}
	departmentType := enum.DepartmentType(httpx.Trim(r.URL.Query().Get("type")))
	user, _ := httpx.MiniAppUserFromContext(r.Context())
	if httpx.IsShopUser(user) {
		if departmentType != "" && departmentType != enum.DepartmentTypeShop {
			httpx.WriteJSON(w, http.StatusOK, []httpx.DepartmentResponse{})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, []httpx.DepartmentResponse{{
			ID:   user.DepartmentID,
			Code: user.DepartmentCode,
			Name: user.DepartmentName,
			Type: user.DepartmentType,
		}})
		return
	}

	departments, err := h.departmentSvc.ListByType(r.Context(), departmentType)
	if err != nil {
		slog.ErrorContext(r.Context(), "list departments failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось получить список магазинов.")
		return
	}
	response := make([]httpx.DepartmentResponse, 0, len(departments))
	for _, department := range departments {
		response = append(response, httpx.DepartmentResponse{
			ID:   department.ID,
			Code: department.Code,
			Name: department.Name,
			Type: department.Type,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}
