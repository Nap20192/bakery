package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	adminuc "bakery/internal/services/admin/usecase/admin"
)

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
	Role           *string `json:"role"`
	DepartmentCode *string `json:"department_code"`
	Password       *string `json:"password"`
}

func (s *Server) handleAdminDepartments(w http.ResponseWriter, r *http.Request) {
	departments, err := s.adminSvc.ListDepartments(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "admin list departments failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось получить отделы."})
		return
	}
	out := make([]departmentResponse, 0, len(departments))
	for _, d := range departments {
		out = append(out, departmentResponse{ID: d.ID, Code: d.Code, Name: d.Name, Type: d.Type})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.adminSvc.ListUsers(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list users failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось получить пользователей."})
		return
	}
	out := make([]userResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toUserResponse(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Некорректные данные."})
		return
	}
	if strings.TrimSpace(req.Username) == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Укажите логин и пароль."})
		return
	}

	user, err := s.adminSvc.CreateUser(r.Context(), adminuc.CreateUserInput{
		Username:         req.Username,
		Password:         req.Password,
		TelegramUsername: req.TelegramUsername,
		Role:             req.Role,
		DepartmentCode:   req.DepartmentCode,
	})
	if err != nil {
		s.writeUserError(w, r, err, "Не удалось создать пользователя (возможно, логин занят).")
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(user))
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Некорректный id пользователя."})
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Некорректные данные."})
		return
	}

	var (
		user    adminuc.User
		updated bool
	)
	if req.Role != nil {
		user, err = s.adminSvc.SetUserRole(r.Context(), id, *req.Role)
		if err != nil {
			s.writeUserError(w, r, err, "Не удалось обновить роль.")
			return
		}
		updated = true
	}
	if req.DepartmentCode != nil {
		user, err = s.adminSvc.AssignUserDepartment(r.Context(), id, *req.DepartmentCode)
		if err != nil {
			s.writeUserError(w, r, err, "Не удалось обновить отдел.")
			return
		}
		updated = true
	}
	if req.Password != nil {
		user, err = s.adminSvc.SetUserPassword(r.Context(), id, *req.Password)
		if err != nil {
			s.writeUserError(w, r, err, "Не удалось обновить пароль.")
			return
		}
		updated = true
	}
	if !updated {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Нечего обновлять."})
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Некорректный id пользователя."})
		return
	}
	if err := s.adminSvc.DeleteUser(r.Context(), id); err != nil {
		slog.ErrorContext(r.Context(), "delete user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось удалить пользователя."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) writeUserError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	switch {
	case errors.Is(err, adminuc.ErrInvalidRole):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Недопустимая роль."})
	default:
		slog.ErrorContext(r.Context(), "admin user operation failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fallback})
	}
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
