package adminuc

import (
	"context"
	"fmt"
	"strings"

	"bakery/internal/pkg/apperr"
	accessdomain "bakery/internal/services/auth/domain"
)

var ErrInvalidRole = apperr.Invalid("admin.invalid_role", "Недопустимая роль.")

// Service is the admin use-case implementation. It depends only on the
// UserAccounts and Departments ports.
type Service struct {
	users       UserAccounts
	departments Departments
}

var _ UseCase = (*Service)(nil)

func NewService(users UserAccounts, departments Departments) *Service {
	return &Service{users: users, departments: departments}
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.users.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toUser(u))
	}
	return out, nil
}

func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (User, error) {
	role := accessdomain.NormalizeRole(input.Role)
	if !accessdomain.IsValidRole(role) {
		return User{}, fmt.Errorf("%w: %s", ErrInvalidRole, input.Role)
	}
	departmentID, err := s.resolveDepartment(ctx, input.DepartmentCode)
	if err != nil {
		return User{}, err
	}
	user, err := s.users.CreateUserWithPassword(ctx, accessdomain.PasswordAuthUserInput{
		Username:         strings.TrimSpace(input.Username),
		Password:         input.Password,
		TelegramUsername: normalizeTelegramUsername(input.TelegramUsername),
		Role:             role,
		DepartmentID:     departmentID,
	})
	if err != nil {
		return User{}, err
	}
	return toUser(user), nil
}

func (s *Service) SetUserRole(ctx context.Context, id int64, role string) (User, error) {
	normalized := accessdomain.NormalizeRole(role)
	if !accessdomain.IsValidRole(normalized) {
		return User{}, fmt.Errorf("%w: %s", ErrInvalidRole, role)
	}
	user, err := s.users.SetUserRole(ctx, id, normalized)
	if err != nil {
		return User{}, err
	}
	return toUser(user), nil
}

func (s *Service) SetUserUsername(ctx context.Context, id int64, username string) (User, error) {
	user, err := s.users.SetUsername(ctx, id, strings.TrimSpace(username))
	if err != nil {
		return User{}, err
	}
	return toUser(user), nil
}

func (s *Service) AssignUserDepartment(ctx context.Context, id int64, departmentCode string) (User, error) {
	departmentID, err := s.resolveDepartment(ctx, departmentCode)
	if err != nil {
		return User{}, err
	}
	user, err := s.users.AssignUserDepartment(ctx, id, departmentID)
	if err != nil {
		return User{}, err
	}
	return toUser(user), nil
}

func (s *Service) SetUserPassword(ctx context.Context, id int64, password string) (User, error) {
	if password == "" {
		return User{}, fmt.Errorf("password is required")
	}
	user, err := s.users.SetPassword(ctx, id, password)
	if err != nil {
		return User{}, err
	}
	return toUser(user), nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	return s.users.DeleteUser(ctx, id)
}

func (s *Service) ListDepartments(ctx context.Context) ([]Department, error) {
	return s.departments.ListAll(ctx)
}

// resolveDepartment maps an optional department code to its id (nil = none).
func (s *Service) resolveDepartment(ctx context.Context, code string) (*int64, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	department, err := s.departments.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("department code %q not found: %w", code, err)
	}
	id := department.ID
	return &id, nil
}

// normalizeTelegramUsername trims a leading @ and surrounding spaces so it
// matches the username Telegram reports for the sender.
func normalizeTelegramUsername(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "@")
}

func toUser(u accessdomain.AuthUser) User {
	telegramUsername := ""
	if u.TelegramUsername != nil {
		telegramUsername = *u.TelegramUsername
	}
	return User{
		ID:               u.ID,
		Username:         u.Username,
		TelegramUsername: telegramUsername,
		TelegramID:       u.TelegramID,
		Role:             u.Role,
		DepartmentID:     u.DepartmentID,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}
