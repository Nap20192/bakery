// Package authuc is the application (use-case) layer of the auth service.
//
// It owns the boundary contract (UseCase), the RBAC policy, and the persistence
// port (Repository). Following dependency inversion, this inner layer declares
// the interface it needs; the infra adapter (infra/repo) implements it and
// depends on this package, never the other way around.
package authuc

import (
	"context"

	accessdomain "bakery/internal/domain/access"
)

// UseCase is the auth boundary used by delivery (bot, HTTP API) and bootstrap.
type UseCase interface {
	CreateUserWithPassword(ctx context.Context, input accessdomain.PasswordAuthUserInput) (accessdomain.AuthUser, error)
	EnsureAdminUser(ctx context.Context, username, password string) (accessdomain.AuthUser, bool, error)
	VerifyPassword(ctx context.Context, username, password string) (accessdomain.AuthUser, error)
	LoginTelegramUser(ctx context.Context, telegramID int64, telegramUsername, username, password string) (accessdomain.AuthUser, error)
	LogoutTelegramUser(ctx context.Context, telegramID int64) error
	AssignUserDepartment(ctx context.Context, userID int64, departmentID *int64) (accessdomain.AuthUser, error)
	SetTelegramUserDepartment(ctx context.Context, telegramID int64, telegramUsername string, departmentID int64) (accessdomain.AuthUser, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error)
	GetUserByTelegramUsername(ctx context.Context, telegramUsername string) (accessdomain.AuthUser, error)
	ListUsersByDepartmentID(ctx context.Context, departmentID int64) ([]accessdomain.AuthUser, error)
	ListUsersByRole(ctx context.Context, role string) ([]accessdomain.AuthUser, error)
}

// Repository is the persistence port implemented by infra/repo. It owns row
// timestamps; the use case deals only in domain values.
type Repository interface {
	CreatePasswordUser(ctx context.Context, input CreatePasswordUserInput) (accessdomain.AuthUser, error)
	// GetByUsername returns the user and their stored password hash (the hash is
	// kept out of the domain model and only surfaced here for verification).
	GetByUsername(ctx context.Context, username string) (accessdomain.AuthUser, string, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error)
	GetByTelegramUsername(ctx context.Context, telegramUsername string) (accessdomain.AuthUser, error)
	ListByDepartmentID(ctx context.Context, departmentID int64) ([]accessdomain.AuthUser, error)
	ListByRole(ctx context.Context, role string) ([]accessdomain.AuthUser, error)
	LinkTelegramUser(ctx context.Context, input LinkTelegramUserInput) (accessdomain.AuthUser, error)
	UnlinkTelegramUser(ctx context.Context, telegramID int64) error
	UpsertTelegramUserDepartment(ctx context.Context, input UpsertTelegramDepartmentInput) (accessdomain.AuthUser, error)
	AssignUserDepartment(ctx context.Context, userID int64, departmentID *int64) (accessdomain.AuthUser, error)
}

type CreatePasswordUserInput struct {
	DepartmentID *int64
	Username     string
	PasswordHash string
	MetadataJSON string
	Role         string
}

type LinkTelegramUserInput struct {
	UserID           int64
	TelegramID       int64
	TelegramUsername string
}

type UpsertTelegramDepartmentInput struct {
	TelegramID       int64
	TelegramUsername string
	DepartmentID     int64
}
