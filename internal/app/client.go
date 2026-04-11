package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"bakery/internal/domain"
	"bakery/internal/adapter/outbound/postgres"
)

type ClientService struct {
	q *postgres.Queries
}

func NewClientService(q *postgres.Queries) *ClientService {
	return &ClientService{q: q}
}

type CreateClientInput struct {
	Name           string
	TelegramChatID int64
	Phone          string
	Note           string
}

func (s *ClientService) Create(ctx context.Context, in CreateClientInput) (domain.Client, error) {
	if in.Name == "" {
		return domain.Client{}, fmt.Errorf("client name is required")
	}

	row, err := s.q.ClientInsert(ctx, postgres.ClientInsertParams{
		Name:           in.Name,
		TelegramChatID: int64Ptr(in.TelegramChatID),
		Phone:          strPtr(in.Phone),
		Note:           strPtr(in.Note),
	})
	if err != nil {
		return domain.Client{}, fmt.Errorf("create client: %w", err)
	}
	return clientToDomain(row), nil
}

func (s *ClientService) GetByID(ctx context.Context, id uuid.UUID) (domain.Client, error) {
	row, err := s.q.ClientGetByID(ctx, id)
	if err != nil {
		return domain.Client{}, fmt.Errorf("get client %s: %w", id, err)
	}
	return clientToDomain(row), nil
}

func (s *ClientService) GetByTelegramChatID(ctx context.Context, chatID int64) (domain.Client, error) {
	row, err := s.q.ClientGetByTelegramChatID(ctx, &chatID)
	if err != nil {
		return domain.Client{}, fmt.Errorf("get client by telegram %d: %w", chatID, err)
	}
	return clientToDomain(row), nil
}

func (s *ClientService) List(ctx context.Context) ([]domain.Client, error) {
	rows, err := s.q.ClientList(ctx)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	out := make([]domain.Client, len(rows))
	for i, r := range rows {
		out[i] = clientToDomain(r)
	}
	return out, nil
}

type UpdateClientInput struct {
	ID    uuid.UUID
	Name  string
	Phone string
	Note  string
}

func (s *ClientService) Update(ctx context.Context, in UpdateClientInput) (domain.Client, error) {
	if in.Name == "" {
		return domain.Client{}, fmt.Errorf("client name is required")
	}

	row, err := s.q.ClientUpdate(ctx, postgres.ClientUpdateParams{
		ID:    in.ID,
		Name:  in.Name,
		Phone: strPtr(in.Phone),
		Note:  strPtr(in.Note),
	})
	if err != nil {
		return domain.Client{}, fmt.Errorf("update client %s: %w", in.ID, err)
	}
	return clientToDomain(row), nil
}

func (s *ClientService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.ClientSoftDelete(ctx, id); err != nil {
		return fmt.Errorf("delete client %s: %w", id, err)
	}
	return nil
}

func clientToDomain(r postgres.Client) domain.Client {
	return domain.Client{
		ID:             r.ID,
		Name:           r.Name,
		TelegramChatID: derefInt64(r.TelegramChatID),
		Phone:          derefStr(r.Phone),
		Note:           derefStr(r.Note),
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
		DeletedAt:      r.DeletedAt,
	}
}
