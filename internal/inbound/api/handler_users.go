package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	accessdomain "bakery/internal/domain/access"
	authuc "bakery/internal/services/auth/usecase/auth"
)

type userResponse struct {
	ID               int64  `json:"id"`
	Username         string `json:"username"`
	TelegramUsername string `json:"telegram_username"`
	Role             string `json:"role"`
	DepartmentID     *int64 `json:"department_id"`
}

type createUserRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	Role           string `json:"role"`
	DepartmentCode string `json:"department_code"`
}

type updateUserRequest struct {
	Role           *string `json:"role"`
	DepartmentCode *string `json:"department_code"`
}

// handleAdminDepartments returns all departments (admin scope, unfiltered).
func (s *Server) handleAdminDepartments(w http.ResponseWriter, r *http.Request) {
	if s.departmentSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис отделов недоступен."})
		return
	}
	departments, err := s.departmentSvc.ListByType(r.Context(), "")
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
	users, err := s.authSvc.ListUsers(r.Context())
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
	req.Username = strings.TrimSpace(req.Username)
	req.Role = accessdomain.NormalizeRole(req.Role)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Укажите логин и пароль."})
		return
	}
	if !accessdomain.IsValidRole(req.Role) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Недопустимая роль."})
		return
	}

	departmentID, apiErr := s.departmentIDByCode(r, req.DepartmentCode)
	if apiErr != nil {
		writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
		return
	}

	user, err := s.authSvc.CreateUserWithPassword(r.Context(), accessdomain.PasswordAuthUserInput{
		Username:     req.Username,
		Password:     req.Password,
		Role:         req.Role,
		DepartmentID: departmentID,
	})
	if err != nil {
		if errors.Is(err, authuc.ErrInvalidRole) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Недопустимая роль."})
			return
		}
		slog.ErrorContext(r.Context(), "create user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось создать пользователя (возможно, логин занят)."})
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

	user := accessdomain.AuthUser{}
	updated := false
	if req.Role != nil {
		role := accessdomain.NormalizeRole(*req.Role)
		if !accessdomain.IsValidRole(role) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Недопустимая роль."})
			return
		}
		user, err = s.authSvc.SetUserRole(r.Context(), id, role)
		if err != nil {
			s.writeUserUpdateError(w, r, err)
			return
		}
		updated = true
	}
	if req.DepartmentCode != nil {
		departmentID, apiErr := s.departmentIDByCode(r, *req.DepartmentCode)
		if apiErr != nil {
			writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
			return
		}
		user, err = s.authSvc.AssignUserDepartment(r.Context(), id, departmentID)
		if err != nil {
			s.writeUserUpdateError(w, r, err)
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

func (s *Server) writeUserUpdateError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, authuc.ErrAuthUserNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "Пользователь не найден."})
		return
	}
	slog.ErrorContext(r.Context(), "update user failed", "error", err)
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось обновить пользователя."})
}

// departmentIDByCode resolves an optional department code to its id.
func (s *Server) departmentIDByCode(r *http.Request, code string) (*int64, *viewerError) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	department, err := s.departmentSvc.GetByCode(r.Context(), code)
	if err != nil {
		return nil, &viewerError{http.StatusBadRequest, "Отдел с таким кодом не найден."}
	}
	id := department.ID
	return &id, nil
}

func toUserResponse(u accessdomain.AuthUser) userResponse {
	telegramUsername := ""
	if u.TelegramUsername != nil {
		telegramUsername = *u.TelegramUsername
	}
	return userResponse{
		ID:               u.ID,
		Username:         u.Username,
		TelegramUsername: telegramUsername,
		Role:             u.Role,
		DepartmentID:     u.DepartmentID,
	}
}
