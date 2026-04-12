package app

import (
	"context"
	"fmt"

	"bakery/internal/domain/client"
	"bakery/internal/domain/user"
)

type ClientService struct {
	clients client.Repository
}

func NewClientService(clients client.Repository) *ClientService {
	return &ClientService{clients: clients}
}

type CreateClientInput struct {
	Name           string
	UserID         *user.ID
	TelegramChatID *int64
	Note           string
}

type UpdateClientInput struct {
	Name           string
	TelegramChatID *int64
	Note           string
}

func (s *ClientService) Create(ctx context.Context, in CreateClientInput) (*client.Client, error) {
	c, err := client.New(in.Name, in.UserID, in.TelegramChatID, in.Note)
	if err != nil {
		return nil, err
	}
	if err := s.clients.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("client: save: %w", err)
	}
	return c, nil
}

func (s *ClientService) GetByID(ctx context.Context, id client.ID) (*client.Client, error) {
	return s.clients.GetByID(ctx, id)
}

func (s *ClientService) List(ctx context.Context) ([]*client.Client, error) {
	return s.clients.List(ctx)
}

func (s *ClientService) Update(ctx context.Context, id client.ID, in UpdateClientInput) (*client.Client, error) {
	c, err := s.clients.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := c.Rename(in.Name); err != nil {
		return nil, err
	}
	c.SetTelegramChatID(in.TelegramChatID)
	if err := s.clients.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("client: save: %w", err)
	}
	return c, nil
}

func (s *ClientService) Delete(ctx context.Context, id client.ID) error {
	return s.clients.Delete(ctx, id)
}
