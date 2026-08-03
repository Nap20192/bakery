package authrepo

import (
	"context"
	"errors"
	"testing"

	sqlc "bakery/internal/outbound/db/sqlc"
	authuc "bakery/internal/services/auth/usecase/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreatePasswordUserMapsIdentityConflicts(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		want       error
	}{
		{
			name:       "username",
			constraint: "idx_auth_users_username",
			want:       authuc.ErrUsernameTaken,
		},
		{
			name:       "telegram username",
			constraint: "idx_auth_users_telegram_username",
			want:       authuc.ErrTelegramUsernameTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := New(sqlc.New(errorDB{
				err: &pgconn.PgError{Code: "23505", ConstraintName: tt.constraint},
			}))

			_, err := repo.CreatePasswordUser(context.Background(), authuc.CreatePasswordUserInput{
				Username:     "existing",
				PasswordHash: "hash",
				MetadataJSON: "{}",
				Role:         "shop",
			})
			if !errors.Is(err, tt.want) {
				t.Fatalf("CreatePasswordUser error = %v, want %v", err, tt.want)
			}
		})
	}
}

type errorDB struct {
	err error
}

func (db errorDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected Exec call")
}

func (db errorDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query call")
}

func (db errorDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return errorRow(db)
}

type errorRow struct {
	err error
}

func (row errorRow) Scan(...interface{}) error {
	return row.err
}
