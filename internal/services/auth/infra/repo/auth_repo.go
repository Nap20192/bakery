// Package authrepo is the persistence adapter of the auth service. It
// implements the authuc.Repository port over sqlc + pgx, maps rows to the
// access domain model, and owns row timestamps. The use case depends on the
// port, not on this package.
package authrepo

import (
	"context"
	"errors"
	"fmt"

	accessdomain "bakery/internal/domain/access"
	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/helpers"
	authuc "bakery/internal/services/auth/usecase/auth"

	"github.com/jackc/pgx/v5"
)

type AuthRepository struct {
	queries *sqlc.Queries
}

var _ authuc.Repository = (*AuthRepository)(nil)

func New(queries *sqlc.Queries) *AuthRepository {
	return &AuthRepository{queries: queries}
}

func (r *AuthRepository) CreatePasswordUser(ctx context.Context, input authuc.CreatePasswordUserInput) (accessdomain.AuthUser, error) {
	now := helpers.TimestamptzNow()
	user, err := r.queries.CreatePasswordAuthUser(ctx, sqlc.CreatePasswordAuthUserParams{
		DepartmentID: input.DepartmentID,
		Username:     input.Username,
		PasswordHash: input.PasswordHash,
		MetadataJson: input.MetadataJSON,
		Role:         input.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("create password auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) GetByUsername(ctx context.Context, username string) (accessdomain.AuthUser, string, error) {
	user, err := r.queries.GetAuthUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, "", authuc.ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, "", fmt.Errorf("get auth user by username: %w", err)
	}
	return authUserToDomain(user), user.PasswordHash, nil
}

func (r *AuthRepository) GetByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error) {
	user, err := r.queries.GetAuthUserByTelegramID(ctx, &telegramID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, authuc.ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("get auth user by telegram id: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) GetByTelegramUsername(ctx context.Context, telegramUsername string) (accessdomain.AuthUser, error) {
	user, err := r.queries.GetAuthUserByTelegramUsername(ctx, telegramUsername)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, authuc.ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("get auth user by telegram username: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) ListByRole(ctx context.Context, role string) ([]accessdomain.AuthUser, error) {
	users, err := r.queries.ListAuthUsersByRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("list auth users by role: %w", err)
	}
	result := make([]accessdomain.AuthUser, 0, len(users))
	for _, user := range users {
		result = append(result, authUserToDomain(user))
	}
	return result, nil
}

func (r *AuthRepository) ListByDepartmentID(ctx context.Context, departmentID int64) ([]accessdomain.AuthUser, error) {
	users, err := r.queries.ListAuthUsersByDepartmentID(ctx, &departmentID)
	if err != nil {
		return nil, fmt.Errorf("list auth users by department: %w", err)
	}
	result := make([]accessdomain.AuthUser, 0, len(users))
	for _, user := range users {
		result = append(result, authUserToDomain(user))
	}
	return result, nil
}

func (r *AuthRepository) LinkTelegramUser(ctx context.Context, input authuc.LinkTelegramUserInput) (accessdomain.AuthUser, error) {
	user, err := r.queries.LinkTelegramAuthUser(ctx, sqlc.LinkTelegramAuthUserParams{
		TelegramID:       &input.TelegramID,
		TelegramUsername: optionalTelegramUsername(input.TelegramUsername),
		UpdatedAt:        helpers.TimestamptzNow(),
		ID:               input.UserID,
	})
	if err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("link telegram auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) UnlinkTelegramUser(ctx context.Context, telegramID int64) error {
	if err := r.queries.UnlinkTelegramAuthUser(ctx, sqlc.UnlinkTelegramAuthUserParams{
		TelegramID: &telegramID,
		UpdatedAt:  helpers.TimestamptzNow(),
	}); err != nil {
		return fmt.Errorf("unlink telegram auth user: %w", err)
	}
	return nil
}

func (r *AuthRepository) UpsertTelegramUserDepartment(ctx context.Context, input authuc.UpsertTelegramDepartmentInput) (accessdomain.AuthUser, error) {
	now := helpers.TimestamptzNow()
	departmentID := input.DepartmentID
	user, err := r.queries.UpsertTelegramAuthUserDepartment(ctx, sqlc.UpsertTelegramAuthUserDepartmentParams{
		TelegramID:       &input.TelegramID,
		TelegramUsername: optionalTelegramUsername(input.TelegramUsername),
		DepartmentID:     &departmentID,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("set telegram user department: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) AssignUserDepartment(ctx context.Context, userID int64, departmentID *int64) (accessdomain.AuthUser, error) {
	user, err := r.queries.AssignUserDepartment(ctx, sqlc.AssignUserDepartmentParams{
		DepartmentID: departmentID,
		UpdatedAt:    helpers.TimestamptzNow(),
		UserID:       userID,
	})
	if err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("assign user department: %w", err)
	}
	return authUserToDomain(user), nil
}

func authUserToDomain(user sqlc.AuthUser) accessdomain.AuthUser {
	return accessdomain.AuthUser{
		ID:               user.ID,
		TelegramID:       user.TelegramID,
		TelegramUsername: user.TelegramUsername,
		DepartmentID:     user.DepartmentID,
		Username:         user.Username,
		MetadataJSON:     user.MetadataJson,
		Role:             accessdomain.NormalizeRole(user.Role),
		CreatedAt:        user.CreatedAt.Time,
		UpdatedAt:        user.UpdatedAt.Time,
	}
}

func optionalTelegramUsername(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
