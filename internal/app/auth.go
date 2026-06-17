package app

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	accessdomain "bakery/internal/domain/access"
	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/helpers"

	"github.com/jackc/pgx/v5"
)

const (
	defaultAuthRole       = accessdomain.RoleAdmin
	passwordHashAlgorithm = "pbkdf2-sha256" //nolint:gosec // algorithm name, not a credential.
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

func (s *AuthService) CreateUserWithPassword(ctx context.Context, input accessdomain.PasswordAuthUserInput) (accessdomain.AuthUser, error) {
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" {
		return accessdomain.AuthUser{}, fmt.Errorf("username is required")
	}
	if input.Password == "" {
		return accessdomain.AuthUser{}, fmt.Errorf("password is required")
	}
	if input.Role == "" {
		input.Role = defaultAuthRole
	}
	input.Role = accessdomain.NormalizeRole(input.Role)
	if !accessdomain.IsValidRole(input.Role) {
		return accessdomain.AuthUser{}, fmt.Errorf("%w: %s", ErrInvalidRole, input.Role)
	}
	if input.MetadataJSON == "" {
		input.MetadataJSON = "{}"
	}

	hash, err := hashPassword(input.Password)
	if err != nil {
		return accessdomain.AuthUser{}, err
	}

	now := helpers.TimestamptzNow()
	user, err := s.queries.CreatePasswordAuthUser(ctx, sqlc.CreatePasswordAuthUserParams{
		DepartmentID: input.DepartmentID,
		Username:     input.Username,
		PasswordHash: hash,
		MetadataJson: input.MetadataJSON,
		Role:         input.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("create password auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) EnsureAdminUser(ctx context.Context, username string, password string) (accessdomain.AuthUser, bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return accessdomain.AuthUser{}, false, fmt.Errorf("admin username is required")
	}
	if password == "" {
		return accessdomain.AuthUser{}, false, fmt.Errorf("admin password is required")
	}

	user, err := s.queries.GetAuthUserByUsername(ctx, username)
	if err == nil {
		return authUserToDomain(user), false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return accessdomain.AuthUser{}, false, fmt.Errorf("get admin user: %w", err)
	}

	created, err := s.CreateUserWithPassword(ctx, accessdomain.PasswordAuthUserInput{
		Username: username,
		Password: password,
		Role:     accessdomain.RoleAdmin,
	})
	if err != nil {
		return accessdomain.AuthUser{}, false, err
	}
	return created, true, nil
}

func (s *AuthService) VerifyPassword(ctx context.Context, username string, password string) (accessdomain.AuthUser, error) {
	user, err := s.queries.GetAuthUserByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("get auth user: %w", err)
	}
	if user.PasswordHash == "" || !verifyPassword(password, user.PasswordHash) {
		return accessdomain.AuthUser{}, fmt.Errorf("invalid credentials")
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) LoginTelegramUser(ctx context.Context, telegramID int64, telegramUsername string, username string, password string) (accessdomain.AuthUser, error) {
	user, err := s.VerifyPassword(ctx, username, password)
	if err != nil {
		return accessdomain.AuthUser{}, err
	}

	now := helpers.TimestamptzNow()
	telegramUsername = strings.TrimSpace(telegramUsername)
	if err := s.queries.UnlinkTelegramAuthUser(ctx, sqlc.UnlinkTelegramAuthUserParams{
		TelegramID: &telegramID,
		UpdatedAt:  now,
	}); err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("unlink previous telegram auth user: %w", err)
	}
	row, err := s.queries.LinkTelegramAuthUser(ctx, sqlc.LinkTelegramAuthUserParams{
		TelegramID:       &telegramID,
		TelegramUsername: optionalTelegramUsername(telegramUsername),
		UpdatedAt:        now,
		ID:               user.ID,
	})
	if err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("link telegram auth user: %w", err)
	}
	return authUserToDomain(row), nil
}

func (s *AuthService) LogoutTelegramUser(ctx context.Context, telegramID int64) error {
	now := helpers.TimestamptzNow()
	if err := s.queries.UnlinkTelegramAuthUser(ctx, sqlc.UnlinkTelegramAuthUserParams{
		TelegramID: &telegramID,
		UpdatedAt:  now,
	}); err != nil {
		return fmt.Errorf("unlink telegram auth user: %w", err)
	}
	return nil
}

func (s *AuthService) AssignUserDepartment(ctx context.Context, userID int64, departmentID *int64) (accessdomain.AuthUser, error) {
	now := helpers.TimestamptzNow()
	user, err := s.queries.AssignUserDepartment(ctx, sqlc.AssignUserDepartmentParams{
		DepartmentID: departmentID,
		UpdatedAt:    now,
		UserID:       userID,
	})
	if err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("assign user department: %w", err)
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) SetTelegramUserDepartment(ctx context.Context, telegramID int64, telegramUsername string, departmentID int64) (accessdomain.AuthUser, error) {
	now := helpers.TimestamptzNow()
	telegramUsername = strings.TrimSpace(telegramUsername)
	user, err := s.queries.UpsertTelegramAuthUserDepartment(ctx, sqlc.UpsertTelegramAuthUserDepartmentParams{
		TelegramID:       &telegramID,
		TelegramUsername: optionalTelegramUsername(telegramUsername),
		DepartmentID:     &departmentID,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return accessdomain.AuthUser{}, fmt.Errorf("set telegram user department: %w", err)
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) GetUserByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error) {
	user, err := s.queries.GetAuthUserByTelegramID(ctx, &telegramID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessdomain.AuthUser{}, ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("get auth user: %w", err)
	}
	return authUserToDomain(user), nil
}

func (s *AuthService) ListUsersByDepartmentID(ctx context.Context, departmentID int64) ([]accessdomain.AuthUser, error) {
	users, err := s.queries.ListAuthUsersByDepartmentID(ctx, &departmentID)
	if err != nil {
		return nil, fmt.Errorf("list auth users by department: %w", err)
	}
	result := make([]accessdomain.AuthUser, 0, len(users))
	for _, user := range users {
		result = append(result, authUserToDomain(user))
	}
	return result, nil
}

func authUserToDomain(user sqlc.AuthUser) accessdomain.AuthUser {
	role := accessdomain.NormalizeRole(user.Role)
	return accessdomain.AuthUser{
		ID:               user.ID,
		TelegramID:       user.TelegramID,
		TelegramUsername: user.TelegramUsername,
		DepartmentID:     user.DepartmentID,
		Username:         user.Username,
		MetadataJSON:     user.MetadataJson,
		Role:             role,
		CreatedAt:        user.CreatedAt.Time,
		UpdatedAt:        user.UpdatedAt.Time,
	}
}

func optionalTelegramUsername(value string) *string {
	if value == "" {
		return nil
	}
	return &value
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
