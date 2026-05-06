package domain

import "time"

type AuthUser struct {
	ID           int64
	TelegramID   *int64
	Username     string
	MetadataJSON string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AuthUserInput struct {
	TelegramID   int64
	Username     string
	MetadataJSON string
	Role         string
}

type PasswordAuthUserInput struct {
	Username     string
	Password     string
	MetadataJSON string
	Role         string
}
