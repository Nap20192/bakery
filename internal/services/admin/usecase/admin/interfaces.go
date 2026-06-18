// Package adminuc is the application layer of the admin service: user
// provisioning, role/department assignment and department listing. It composes
// the auth and department capabilities behind ports (dependency inversion);
// the composition root binds them.
package adminuc

import (
	"context"
	"time"

	accessdomain "bakery/internal/services/auth/domain"
)

// UseCase is the admin boundary used by the HTTP API (admin-only routes).
type UseCase interface {
	ListUsers(ctx context.Context) ([]User, error)
	CreateUser(ctx context.Context, input CreateUserInput) (User, error)
	SetUserRole(ctx context.Context, id int64, role string) (User, error)
	SetUserPassword(ctx context.Context, id int64, password string) (User, error)
	AssignUserDepartment(ctx context.Context, id int64, departmentCode string) (User, error)
	DeleteUser(ctx context.Context, id int64) error
	ListDepartments(ctx context.Context) ([]Department, error)
}

// UserAccounts is the port for user persistence/auth operations (satisfied by
// the auth service).
type UserAccounts interface {
	CreateUserWithPassword(ctx context.Context, input accessdomain.PasswordAuthUserInput) (accessdomain.AuthUser, error)
	ListUsers(ctx context.Context) ([]accessdomain.AuthUser, error)
	SetUserRole(ctx context.Context, id int64, role string) (accessdomain.AuthUser, error)
	SetPassword(ctx context.Context, id int64, password string) (accessdomain.AuthUser, error)
	AssignUserDepartment(ctx context.Context, id int64, departmentID *int64) (accessdomain.AuthUser, error)
	DeleteUser(ctx context.Context, id int64) error
}

// Departments is the port for department lookup (satisfied by the department
// service via an adapter).
type Departments interface {
	GetByCode(ctx context.Context, code string) (Department, error)
	ListAll(ctx context.Context) ([]Department, error)
}

type User struct {
	ID               int64
	Username         string
	TelegramUsername string
	TelegramID       *int64
	Role             string
	DepartmentID     *int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Department struct {
	ID   int64
	Code string
	Name string
	Type string
}

type CreateUserInput struct {
	Username         string
	Password         string
	TelegramUsername string
	Role             string
	DepartmentCode   string
}
