// Package authuc is the application (use-case) layer of the auth service.
//
// It owns the boundary contract (UseCase), the RBAC policy, and the persistence
// port (Repository). Following dependency inversion, this inner layer declares
// the interface it needs; the infra adapter (infra/repo) implements it and
// depends on this package, never the other way around.
package authuc

import (
	"context"

	accessdomain "bakery/internal/services/auth/domain"
)

// UseCase is the auth boundary used by delivery (bot, HTTP API) and bootstrap.
type UseCase interface {
	CreateUserWithPassword(ctx context.Context, input accessdomain.PasswordAuthUserInput) (accessdomain.AuthUser, error)
	EnsureAdminUser(ctx context.Context, username, password string) (accessdomain.AuthUser, bool, error)
	VerifyPassword(ctx context.Context, username, password string) (accessdomain.AuthUser, error)
	// AuthenticateTelegram binds the given telegram_id to the account whose
	// telegram_username matches.
	AuthenticateTelegram(ctx context.Context, telegramID int64, telegramUsername string) (accessdomain.AuthUser, error)
	// BindTelegram binds telegram_id to a known account directly — used by the
	// /login fallback after the password proved ownership.
	BindTelegram(ctx context.Context, userID, telegramID int64) (accessdomain.AuthUser, error)
	SetPassword(ctx context.Context, id int64, password string) (accessdomain.AuthUser, error)
	DeleteUser(ctx context.Context, id int64) error
	AssignUserDepartment(ctx context.Context, userID int64, departmentID *int64) (accessdomain.AuthUser, error)
	GetUserByID(ctx context.Context, id int64) (accessdomain.AuthUser, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error)
	GetUserByTelegramUsername(ctx context.Context, telegramUsername string) (accessdomain.AuthUser, error)
	ListUsers(ctx context.Context) ([]accessdomain.AuthUser, error)
	ListUsersByDepartmentID(ctx context.Context, departmentID int64) ([]accessdomain.AuthUser, error)
	ListUsersByRole(ctx context.Context, role string) ([]accessdomain.AuthUser, error)
	SetUserRole(ctx context.Context, id int64, role string) (accessdomain.AuthUser, error)
	SetUsername(ctx context.Context, id int64, username string) (accessdomain.AuthUser, error)
}

// Repository is the persistence port implemented by infra/repo. It owns row
// timestamps; the use case deals only in domain values.
type Repository interface {
	CreatePasswordUser(ctx context.Context, input CreatePasswordUserInput) (accessdomain.AuthUser, error)
	// GetByUsername returns the user and their stored password hash (the hash is
	// kept out of the domain model and only surfaced here for verification).
	GetByUsername(ctx context.Context, username string) (accessdomain.AuthUser, string, error)
	GetByID(ctx context.Context, id int64) (accessdomain.AuthUser, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error)
	GetByTelegramUsername(ctx context.Context, telegramUsername string) (accessdomain.AuthUser, error)
	ListAll(ctx context.Context) ([]accessdomain.AuthUser, error)
	ListByDepartmentID(ctx context.Context, departmentID int64) ([]accessdomain.AuthUser, error)
	ListByRole(ctx context.Context, role string) ([]accessdomain.AuthUser, error)
	SetRole(ctx context.Context, id int64, role string) (accessdomain.AuthUser, error)
	SetUsername(ctx context.Context, id int64, username string) (accessdomain.AuthUser, error)
	SetPasswordHash(ctx context.Context, id int64, passwordHash string) (accessdomain.AuthUser, error)
	BindTelegramID(ctx context.Context, id, telegramID int64) (accessdomain.AuthUser, error)
	Delete(ctx context.Context, id int64) error
	AssignUserDepartment(ctx context.Context, userID int64, departmentID *int64) (accessdomain.AuthUser, error)
}

type CreatePasswordUserInput struct {
	DepartmentID     *int64
	Username         string
	TelegramUsername string
	PasswordHash     string
	MetadataJSON     string
	Role             string
}
