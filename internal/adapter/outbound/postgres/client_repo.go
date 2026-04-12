package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	db "bakery/internal/adapter/outbound/db"
	"bakery/internal/domain/client"
	"bakery/internal/domain/user"
)

type ClientRepository struct {
	q *db.Queries
}

func NewClientRepository(q *db.Queries) *ClientRepository {
	return &ClientRepository{q: q}
}

func (r *ClientRepository) Save(ctx context.Context, c *client.Client) error {
	_, err := r.q.GetClientByID(ctx, c.ID())
	note := nullString(c.Note())
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if _, err := r.q.CreateClient(ctx, db.CreateClientParams{
			ID:             c.ID(),
			UserID:         c.UserID(),
			Name:           c.Name(),
			TelegramChatID: c.TelegramChatID(),
			Note:           note,
		}); err != nil {
			return fmt.Errorf("client repo: create: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("client repo: probe: %w", err)
	}
	if _, err := r.q.UpdateClient(ctx, db.UpdateClientParams{
		ID:             c.ID(),
		Name:           c.Name(),
		TelegramChatID: c.TelegramChatID(),
		Note:           note,
	}); err != nil {
		return fmt.Errorf("client repo: update: %w", err)
	}
	return nil
}

func (r *ClientRepository) GetByID(ctx context.Context, id client.ID) (*client.Client, error) {
	row, err := r.q.GetClientByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, client.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("client repo: get by id: %w", err)
	}
	return toClient(row), nil
}

func (r *ClientRepository) GetByUserID(ctx context.Context, userID user.ID) (*client.Client, error) {
	uid := userID
	row, err := r.q.GetClientByUserID(ctx, &uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, client.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("client repo: get by user id: %w", err)
	}
	return toClient(row), nil
}

func (r *ClientRepository) GetByTelegramChatID(ctx context.Context, chatID int64) (*client.Client, error) {
	cid := chatID
	row, err := r.q.GetClientByTelegramChatID(ctx, &cid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, client.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("client repo: get by telegram: %w", err)
	}
	return toClient(row), nil
}

func (r *ClientRepository) List(ctx context.Context) ([]*client.Client, error) {
	rows, err := r.q.ListClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("client repo: list: %w", err)
	}
	out := make([]*client.Client, 0, len(rows))
	for _, row := range rows {
		out = append(out, toClient(row))
	}
	return out, nil
}

func (r *ClientRepository) Delete(ctx context.Context, id client.ID) error {
	if err := r.q.DeleteClient(ctx, id); err != nil {
		return fmt.Errorf("client repo: delete: %w", err)
	}
	return nil
}

func toClient(row db.Client) *client.Client {
	var note string
	if row.Note != nil {
		note = *row.Note
	}
	return client.Restore(
		row.ID,
		row.UserID,
		row.Name,
		row.TelegramChatID,
		note,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
