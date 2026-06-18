// Package adminhttp is the HTTP delivery adapter of the admin service: user
// management and the admin department list.
package adminhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bakery/internal/inbound/api/httpx"
	adminuc "bakery/internal/services/admin/usecase/admin"
)

// Handler is the admin service HTTP delivery adapter.
type Handler struct {
	adminSvc adminuc.UseCase
}

func New(adminSvc adminuc.UseCase) *Handler {
	return &Handler{adminSvc: adminSvc}
}

// RegisterRoutes wires user management endpoints behind the admin role.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("GET /users", auth.RequireAdmin(http.HandlerFunc(h.handleListUsers)))
	mux.Handle("POST /users", auth.RequireAdmin(http.HandlerFunc(h.handleCreateUser)))
	mux.Handle("PATCH /users/{id}", auth.RequireAdmin(http.HandlerFunc(h.handleUpdateUser)))
	mux.Handle("DELETE /users/{id}", auth.RequireAdmin(http.HandlerFunc(h.handleDeleteUser)))
	mux.Handle("GET /admin/departments", auth.RequireAdmin(http.HandlerFunc(h.handleAdminDepartments)))
}

type userResponse struct {
	ID               int64  `json:"id"`
	Username         string `json:"username"`
	TelegramUsername string `json:"telegram_username"`
	TelegramID       *int64 `json:"telegram_id"`
	Role             string `json:"role"`
	DepartmentID     *int64 `json:"department_id"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type createUserRequest struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	TelegramUsername string `json:"telegram_username"`
	Role             string `json:"role"`
	DepartmentCode   string `json:"department_code"`
}

type updateUserRequest struct {
	Username       *string `json:"username"`
	Role           *string `json:"role"`
	DepartmentCode *string `json:"department_code"`
	Password       *string `json:"password"`
}

func (h *Handler) handleAdminDepartments(w http.ResponseWriter, r *http.Request) {
	departments, err := h.adminSvc.ListDepartments(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "admin list departments failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось получить отделы.")
		return
	}
	out := make([]httpx.DepartmentResponse, 0, len(departments))
	for _, d := range departments {
		out = append(out, httpx.DepartmentResponse{ID: d.ID, Code: d.Code, Name: d.Name, Type: d.Type})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.adminSvc.ListUsers(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list users failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось получить пользователей.")
		return
	}
	out := make([]userResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toUserResponse(u))
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	if strings.TrimSpace(req.Username) == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите логин и пароль.")
		return
	}

	user, err := h.adminSvc.CreateUser(r.Context(), adminuc.CreateUserInput{
		Username:         req.Username,
		Password:         req.Password,
		TelegramUsername: req.TelegramUsername,
		Role:             req.Role,
		DepartmentCode:   req.DepartmentCode,
	})
	if err != nil {
		h.writeUserError(w, r, err, "Не удалось создать пользователя (возможно, логин занят).")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toUserResponse(user))
}

func (h *Handler) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректный id пользователя.")
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}

	var (
		user    adminuc.User
		updated bool
	)
	if req.Username != nil {
		user, err = h.adminSvc.SetUserUsername(r.Context(), id, *req.Username)
		if err != nil {
			h.writeUserError(w, r, err, "Не удалось обновить логин.")
			return
		}
		updated = true
	}
	if req.Role != nil {
		user, err = h.adminSvc.SetUserRole(r.Context(), id, *req.Role)
		if err != nil {
			h.writeUserError(w, r, err, "Не удалось обновить роль.")
			return
		}
		updated = true
	}
	if req.DepartmentCode != nil {
		user, err = h.adminSvc.AssignUserDepartment(r.Context(), id, *req.DepartmentCode)
		if err != nil {
			h.writeUserError(w, r, err, "Не удалось обновить отдел.")
			return
		}
		updated = true
	}
	if req.Password != nil {
		user, err = h.adminSvc.SetUserPassword(r.Context(), id, *req.Password)
		if err != nil {
			h.writeUserError(w, r, err, "Не удалось обновить пароль.")
			return
		}
		updated = true
	}
	if !updated {
		httpx.WriteError(w, http.StatusBadRequest, "Нечего обновлять.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

func (h *Handler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректный id пользователя.")
		return
	}
	if err := h.adminSvc.DeleteUser(r.Context(), id); err != nil {
		slog.ErrorContext(r.Context(), "delete user failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось удалить пользователя.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeUserError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	httpx.WriteAppError(w, r, err, fallback)
}

func toUserResponse(u adminuc.User) userResponse {
	resp := userResponse{
		ID:               u.ID,
		Username:         u.Username,
		TelegramUsername: u.TelegramUsername,
		TelegramID:       u.TelegramID,
		Role:             u.Role,
		DepartmentID:     u.DepartmentID,
	}
	if !u.CreatedAt.IsZero() {
		resp.CreatedAt = u.CreatedAt.Format(time.RFC3339)
	}
	if !u.UpdatedAt.IsZero() {
		resp.UpdatedAt = u.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
