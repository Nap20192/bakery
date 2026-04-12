package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ID = uuid.UUID

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleClient  Role = "client"
	RoleKitchen Role = "kitchen"
)

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleClient || r == RoleKitchen
}

var (
	ErrLoginRequired    = errors.New("user: login is required")
	ErrPasswordRequired = errors.New("user: password hash is required")
	ErrInvalidRole      = errors.New("user: invalid role")
	ErrNotFound         = errors.New("user: not found")
)

type User struct {
	id           ID
	login        string
	passwordHash string
	role         Role
	createdAt    time.Time
	updatedAt    time.Time
}

func New(login, passwordHash string, role Role) (*User, error) {
	if login == "" {
		return nil, ErrLoginRequired
	}
	if passwordHash == "" {
		return nil, ErrPasswordRequired
	}
	if !role.Valid() {
		return nil, ErrInvalidRole
	}
	now := time.Now()
	return &User{
		id:           uuid.New(),
		login:        login,
		passwordHash: passwordHash,
		role:         role,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func Restore(id ID, login, passwordHash string, role Role, createdAt, updatedAt time.Time) *User {
	return &User{
		id:           id,
		login:        login,
		passwordHash: passwordHash,
		role:         role,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

func (u *User) ID() ID                  { return u.id }
func (u *User) Login() string           { return u.login }
func (u *User) PasswordHash() string    { return u.passwordHash }
func (u *User) Role() Role              { return u.role }
func (u *User) CreatedAt() time.Time    { return u.createdAt }
func (u *User) UpdatedAt() time.Time    { return u.updatedAt }

func (u *User) ChangePassword(newHash string) error {
	if newHash == "" {
		return ErrPasswordRequired
	}
	u.passwordHash = newHash
	u.updatedAt = time.Now()
	return nil
}

type Repository interface {
	Save(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id ID) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	Delete(ctx context.Context, id ID) error
}
