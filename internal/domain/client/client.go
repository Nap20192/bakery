package client

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"bakery/internal/domain/user"
)

type ID = uuid.UUID

var (
	ErrNameRequired = errors.New("client: name is required")
	ErrNotFound     = errors.New("client: not found")
)

type Client struct {
	id             ID
	userID         *user.ID
	name           string
	telegramChatID *int64
	note           string
	createdAt      time.Time
	updatedAt      time.Time
}

func New(name string, userID *user.ID, telegramChatID *int64, note string) (*Client, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	now := time.Now()
	return &Client{
		id:             uuid.New(),
		userID:         userID,
		name:           name,
		telegramChatID: telegramChatID,
		note:           note,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

func Restore(id ID, userID *user.ID, name string, telegramChatID *int64, note string, createdAt, updatedAt time.Time) *Client {
	return &Client{
		id:             id,
		userID:         userID,
		name:           name,
		telegramChatID: telegramChatID,
		note:           note,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

func (c *Client) ID() ID                 { return c.id }
func (c *Client) UserID() *user.ID       { return c.userID }
func (c *Client) Name() string           { return c.name }
func (c *Client) TelegramChatID() *int64 { return c.telegramChatID }
func (c *Client) Note() string           { return c.note }
func (c *Client) CreatedAt() time.Time   { return c.createdAt }
func (c *Client) UpdatedAt() time.Time   { return c.updatedAt }

func (c *Client) Rename(name string) error {
	if name == "" {
		return ErrNameRequired
	}
	c.name = name
	c.updatedAt = time.Now()
	return nil
}

func (c *Client) SetTelegramChatID(chatID *int64) {
	c.telegramChatID = chatID
	c.updatedAt = time.Now()
}

type Repository interface {
	Save(ctx context.Context, c *Client) error
	GetByID(ctx context.Context, id ID) (*Client, error)
	GetByUserID(ctx context.Context, userID user.ID) (*Client, error)
	GetByTelegramChatID(ctx context.Context, chatID int64) (*Client, error)
	List(ctx context.Context) ([]*Client, error)
	Delete(ctx context.Context, id ID) error
}
