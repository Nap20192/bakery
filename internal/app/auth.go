package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"bakery/internal/domain"
	sqlcrepo "bakery/internal/repo/sqlc"
)

const defaultAuthRole = "user"

type AuthService struct {
	queries *sqlcrepo.Queries
}

func NewAuthService(queries *sqlcrepo.Queries) *AuthService {
	return &AuthService{queries: queries}
}

func (s *AuthService) CreateOrUpdateUser(ctx context.Context, input domain.AuthUserInput) (domain.AuthUser, error) {
	if input.TelegramID == 0 {
		return domain.AuthUser{}, fmt.Errorf("telegram id is required")
	}
	if input.Role == "" {
		input.Role = defaultAuthRole
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	user, err := s.queries.CreateAuthUser(ctx, sqlcrepo.CreateAuthUserParams{
		TelegramID: input.TelegramID,
		Username:   input.Username,
		Role:       input.Role,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("create auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) GetUserByTelegramID(ctx context.Context, telegramID int64) (domain.AuthUser, error) {
	user, err := s.queries.GetAuthUserByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthUser{}, domain.ErrNotFound
		}
		return domain.AuthUser{}, fmt.Errorf("get auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func authUserToDomain(user sqlcrepo.AuthUser) domain.AuthUser {
	createdAt, _ := time.Parse(time.RFC3339Nano, user.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, user.UpdatedAt)
	return domain.AuthUser{
		ID:         user.ID,
		TelegramID: user.TelegramID,
		Username:   user.Username,
		Role:       user.Role,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}
