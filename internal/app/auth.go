package app

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"bakery/internal/domain"
	sqlc "bakery/internal/repo/sqlc"
)

const (
	defaultAuthRole       = domain.RoleClient
	passwordHashAlgorithm = "pbkdf2-sha256"
	passwordHashVersion   = "v1"
	passwordSaltSize      = 16
	passwordKeySize       = 32
	passwordIterations    = 210000
)

var (
	ErrAuthUserNotFound = errors.New("auth user not found")
	ErrInvalidRole      = errors.New("invalid auth role")
)

type AuthService struct {
	queries *sqlc.Queries
}

func NewAuthService(queries *sqlc.Queries) *AuthService {
	return &AuthService{queries: queries}
}

func (s *AuthService) CreateOrUpdateUser(ctx context.Context, input domain.AuthUserInput) (domain.AuthUser, error) {
	if input.TelegramID == 0 {
		return domain.AuthUser{}, fmt.Errorf("telegram id is required")
	}
	if input.Role == "" {
		input.Role = defaultAuthRole
	}
	input.Role = domain.NormalizeRole(input.Role)
	if !domain.IsValidRole(input.Role) {
		return domain.AuthUser{}, fmt.Errorf("%w: %s", ErrInvalidRole, input.Role)
	}
	if input.MetadataJSON == "" {
		input.MetadataJSON = "{}"
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	telegramID := input.TelegramID
	user, err := s.queries.CreateTelegramAuthUser(ctx, sqlc.CreateTelegramAuthUserParams{
		TelegramID:   &telegramID,
		Username:     input.Username,
		MetadataJson: input.MetadataJSON,
		Role:         input.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("create auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) CreateUserWithPassword(ctx context.Context, input domain.PasswordAuthUserInput) (domain.AuthUser, error) {
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" {
		return domain.AuthUser{}, fmt.Errorf("username is required")
	}
	if input.Password == "" {
		return domain.AuthUser{}, fmt.Errorf("password is required")
	}
	if input.Role == "" {
		input.Role = defaultAuthRole
	}
	input.Role = domain.NormalizeRole(input.Role)
	if !domain.IsValidRole(input.Role) {
		return domain.AuthUser{}, fmt.Errorf("%w: %s", ErrInvalidRole, input.Role)
	}
	if input.MetadataJSON == "" {
		input.MetadataJSON = "{}"
	}

	hash, err := hashPassword(input.Password)
	if err != nil {
		return domain.AuthUser{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	user, err := s.queries.CreatePasswordAuthUser(ctx, sqlc.CreatePasswordAuthUserParams{
		Username:     input.Username,
		PasswordHash: hash,
		MetadataJson: input.MetadataJSON,
		Role:         input.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("create password auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) VerifyPassword(ctx context.Context, username string, password string) (domain.AuthUser, error) {
	user, err := s.queries.GetAuthUserByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthUser{}, ErrAuthUserNotFound
		}
		return domain.AuthUser{}, fmt.Errorf("get auth user: %w", err)
	}
	if user.PasswordHash == "" || !verifyPassword(password, user.PasswordHash) {
		return domain.AuthUser{}, fmt.Errorf("invalid credentials")
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) LoginTelegramUser(ctx context.Context, telegramID int64, username string, password string) (domain.AuthUser, error) {
	user, err := s.VerifyPassword(ctx, username, password)
	if err != nil {
		return domain.AuthUser{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	row, err := s.queries.LinkTelegramAuthUser(ctx, sqlc.LinkTelegramAuthUserParams{
		TelegramID: &telegramID,
		UpdatedAt:  now,
		ID:         user.ID,
	})
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("link telegram auth user: %w", err)
	}
	return authUserToDomain(row), nil
}

func (s *AuthService) GetUserByTelegramID(ctx context.Context, telegramID int64) (domain.AuthUser, error) {
	user, err := s.queries.GetAuthUserByTelegramID(ctx, &telegramID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthUser{}, ErrAuthUserNotFound
		}
		return domain.AuthUser{}, fmt.Errorf("get auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func authUserToDomain(user sqlc.AuthUser) domain.AuthUser {
	createdAt, _ := time.Parse(time.RFC3339Nano, user.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, user.UpdatedAt)
	role := domain.NormalizeRole(user.Role)
	if role == "user" {
		role = domain.RoleClient
	}
	return domain.AuthUser{
		ID:           user.ID,
		TelegramID:   user.TelegramID,
		Username:     user.Username,
		MetadataJSON: user.MetadataJson,
		Role:         role,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeySize)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return strings.Join([]string{
		passwordHashAlgorithm,
		passwordHashVersion,
		fmt.Sprintf("%d", passwordIterations),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	}, "$"), nil
}

func verifyPassword(password string, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != passwordHashAlgorithm || parts[1] != passwordHashVersion {
		return false
	}
	if parts[2] != fmt.Sprintf("%d", passwordIterations) {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	actual, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, len(expected))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
