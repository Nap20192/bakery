// Package authrepo is the persistence adapter of the auth service. It
// implements the authuc.Repository port over sqlc + pgx, maps rows to the
// access domain model, and owns row timestamps. The use case depends on the
// port, not on this package.
package authrepo

import (
	"context"
	"errors"
	"fmt"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/apperr"
	"bakery/internal/pkg/helpers"
	accessdomain "bakery/internal/services/auth/domain"
	authuc "bakery/internal/services/auth/usecase/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		DepartmentID:     input.DepartmentID,
		Username:         input.Username,
		TelegramUsername: optionalTelegramUsername(input.TelegramUsername),
		PasswordHash:     input.PasswordHash,
		MetadataJson:     input.MetadataJSON,
		Role:             input.Role,
		CreatedAt:        now,
		UpdatedAt:        now,
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

func (r *AuthRepository) GetByID(ctx context.Context, id int64) (accessdomain.AuthUser, error) {
	user, err := r.queries.GetAuthUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, authuc.ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("get auth user by id: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) ListAll(ctx context.Context) ([]accessdomain.AuthUser, error) {
	users, err := r.queries.ListAuthUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list auth users: %w", err)
	}
	result := make([]accessdomain.AuthUser, 0, len(users))
	for _, user := range users {
		result = append(result, authUserToDomain(user))
	}
	return result, nil
}

func (r *AuthRepository) SetRole(ctx context.Context, id int64, role string) (accessdomain.AuthUser, error) {
	user, err := r.queries.UpdateAuthUserRole(ctx, sqlc.UpdateAuthUserRoleParams{
		Role:      role,
		UpdatedAt: helpers.TimestamptzNow(),
		ID:        id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, authuc.ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("update auth user role: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) SetUsername(ctx context.Context, id int64, username string) (accessdomain.AuthUser, error) {
	user, err := r.queries.UpdateAuthUserUsername(ctx, sqlc.UpdateAuthUserUsernameParams{
		Username:  username,
		UpdatedAt: helpers.TimestamptzNow(),
		ID:        id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, authuc.ErrAuthUserNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return accessdomain.AuthUser{}, apperr.Conflict("auth.username_taken", "Логин уже занят.")
		}
		return accessdomain.AuthUser{}, fmt.Errorf("update auth user username: %w", err)
	}
	return authUserToDomain(user), nil
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

func (r *AuthRepository) BindTelegramID(ctx context.Context, id, telegramID int64) (accessdomain.AuthUser, error) {
	user, err := r.queries.BindTelegramID(ctx, sqlc.BindTelegramIDParams{
		TelegramID: &telegramID,
		UpdatedAt:  helpers.TimestamptzNow(),
		ID:         id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, authuc.ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("bind telegram id: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) SetPasswordHash(ctx context.Context, id int64, passwordHash string) (accessdomain.AuthUser, error) {
	user, err := r.queries.UpdateAuthUserPassword(ctx, sqlc.UpdateAuthUserPasswordParams{
		PasswordHash: passwordHash,
		UpdatedAt:    helpers.TimestamptzNow(),
		ID:           id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, authuc.ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("update password: %w", err)
	}
	return authUserToDomain(user), nil
}

func (r *AuthRepository) Delete(ctx context.Context, id int64) error {
	if err := r.queries.DeleteAuthUser(ctx, id); err != nil {
		return fmt.Errorf("delete auth user: %w", err)
	}
	return nil
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
