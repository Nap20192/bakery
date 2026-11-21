package authrepo

import (
	"context"
	"errors"
	"testing"

	"bakery/internal/outbound/db/dbtest"
	sqlc "bakery/internal/outbound/db/sqlc"
	authuc "bakery/internal/services/auth/usecase/auth"
)

func TestCreatePasswordUserMapsIdentityConflicts(t *testing.T) {
	tests := []struct {
		name     string
		existing authuc.CreatePasswordUserInput
		create   authuc.CreatePasswordUserInput
		want     error
	}{
		{
			name:     "username",
			existing: authuc.CreatePasswordUserInput{Username: "existing"},
			create:   authuc.CreatePasswordUserInput{Username: "existing"},
			want:     authuc.ErrUsernameTaken,
		},
		{
			name:     "telegram username",
			existing: authuc.CreatePasswordUserInput{Username: "first", TelegramUsername: "baker"},
			create:   authuc.CreatePasswordUserInput{Username: "second", TelegramUsername: "baker"},
			want:     authuc.ErrTelegramUsernameTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := New(sqlc.New(dbtest.Open(t)))
			if _, err := repo.CreatePasswordUser(ctx, withDefaults(tt.existing)); err != nil {
				t.Fatalf("create existing user: %v", err)
			}

			_, err := repo.CreatePasswordUser(ctx, withDefaults(tt.create))
			if !errors.Is(err, tt.want) {
				t.Fatalf("CreatePasswordUser error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestGetByTelegramUsernameReturnsNotFound(t *testing.T) {
	repo := New(sqlc.New(dbtest.Open(t)))
	_, err := repo.GetByTelegramUsername(context.Background(), "nobody")
	if !errors.Is(err, authuc.ErrAuthUserNotFound) {
		t.Fatalf("GetByTelegramUsername error = %v, want %v", err, authuc.ErrAuthUserNotFound)
	}
}

func withDefaults(input authuc.CreatePasswordUserInput) authuc.CreatePasswordUserInput {
	input.PasswordHash = "hash"
	input.MetadataJSON = "{}"
	input.Role = "shop"
	return input
}
