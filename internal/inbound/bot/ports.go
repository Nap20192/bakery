package bot

import (
	"context"

	accessdomain "bakery/internal/services/auth/domain"
	departmentuc "bakery/internal/services/department/usecase/department"
)

// The bot depends on these narrow, bot-owned backend ports rather than on the
// services' full use-case interfaces (interface segregation + dependency
// inversion). The bot is now a thin auth gate (password check via /start) plus
// the order-event notifier to the workshop; it needs only auth and department
// lookups.

// AuthBackend is the slice of the auth service the bot uses.
type AuthBackend interface {
	VerifyPassword(ctx context.Context, username, password string) (accessdomain.AuthUser, error)
	BindTelegram(ctx context.Context, userID, telegramID int64) (accessdomain.AuthUser, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error)
	GetUserByTelegramUsername(ctx context.Context, telegramUsername string) (accessdomain.AuthUser, error)
	ListUsersByRole(ctx context.Context, role string) ([]accessdomain.AuthUser, error)
}

// DepartmentBackend is the slice of the department service the bot uses (for
// rendering order notifications).
type DepartmentBackend interface {
	GetByID(ctx context.Context, id int64) (departmentuc.Department, error)
}
