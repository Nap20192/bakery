package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	db "bakery/internal/adapter/outbound/db"
	"bakery/internal/domain/user"
)

// UserRepository — реализация user.Repository поверх sqlc Queries.
type UserRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) *UserRepository {
	return &UserRepository{q: q}
}

// Save — upsert: insert если нет, иначе обновить password_hash и role.
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	_, err := r.q.GetUserByID(ctx, u.ID())
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		_, err := r.q.CreateUser(ctx, db.CreateUserParams{
			ID:           u.ID(),
			Login:        u.Login(),
			PasswordHash: u.PasswordHash(),
			Role:         string(u.Role()),
		})
		if err != nil {
			return fmt.Errorf("user repo: create: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("user repo: probe: %w", err)
	}
	if _, err := r.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           u.ID(),
		PasswordHash: u.PasswordHash(),
	}); err != nil {
		return fmt.Errorf("user repo: update password: %w", err)
	}
	if _, err := r.q.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:   u.ID(),
		Role: string(u.Role()),
	}); err != nil {
		return fmt.Errorf("user repo: update role: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id user.ID) (*user.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user repo: get by id: %w", err)
	}
	return toUser(row), nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*user.User, error) {
	row, err := r.q.GetUserByLogin(ctx, login)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user repo: get by login: %w", err)
	}
	return toUser(row), nil
}

func (r *UserRepository) Delete(ctx context.Context, id user.ID) error {
	if err := r.q.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("user repo: delete: %w", err)
	}
	return nil
}

func toUser(row db.User) *user.User {
	return user.Restore(
		row.ID,
		row.Login,
		row.PasswordHash,
		user.Role(row.Role),
		row.CreatedAt,
		row.UpdatedAt,
	)
}
