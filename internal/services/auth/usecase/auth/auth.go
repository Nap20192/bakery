package authuc

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

	"bakery/internal/pkg/apperr"
	accessdomain "bakery/internal/services/auth/domain"
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
	ErrAuthUserNotFound      = apperr.NotFound("auth.user_not_found", "Пользователь не найден.")
	ErrInvalidRole           = apperr.Invalid("auth.invalid_role", "Недопустимая роль.")
	ErrUsernameTaken         = apperr.Conflict("auth.username_taken", "Логин уже занят.")
	ErrTelegramUsernameTaken = apperr.Conflict("auth.telegram_username_taken", "Telegram username уже занят.")
)

// Service is the auth use-case implementation. It depends only on the
// Repository port; password hashing is domain policy and lives here.
type Service struct {
	repo Repository
}

var _ UseCase = (*Service)(nil)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateUserWithPassword(ctx context.Context, input accessdomain.PasswordAuthUserInput) (accessdomain.AuthUser, error) {
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
	return s.repo.CreatePasswordUser(ctx, CreatePasswordUserInput{
		DepartmentID:     input.DepartmentID,
		Username:         input.Username,
		TelegramUsername: strings.TrimSpace(input.TelegramUsername),
		PasswordHash:     hash,
		MetadataJSON:     input.MetadataJSON,
		Role:             input.Role,
	})
}

func (s *Service) EnsureAdminUser(ctx context.Context, username, password string) (accessdomain.AuthUser, bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return accessdomain.AuthUser{}, false, fmt.Errorf("admin username is required")
	}
	if password == "" {
		return accessdomain.AuthUser{}, false, fmt.Errorf("admin password is required")
	}

	user, _, err := s.repo.GetByUsername(ctx, username)
	if err == nil {
		return user, false, nil
	}
	if !errors.Is(err, ErrAuthUserNotFound) {
		return accessdomain.AuthUser{}, false, fmt.Errorf("get admin user: %w", err)
	}

	created, err := s.CreateUserWithPassword(ctx, accessdomain.PasswordAuthUserInput{
		Username: username,
		Password: password,
		Role:     accessdomain.RoleAdmin,
	})
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			user, _, getErr := s.repo.GetByUsername(ctx, username)
			if getErr != nil {
				return accessdomain.AuthUser{}, false, fmt.Errorf("get admin user after create conflict: %w", getErr)
			}
			return user, false, nil
		}
		return accessdomain.AuthUser{}, false, err
	}
	return created, true, nil
}

func (s *Service) VerifyPassword(ctx context.Context, username, password string) (accessdomain.AuthUser, error) {
	user, passwordHash, err := s.repo.GetByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, ErrAuthUserNotFound) {
			return accessdomain.AuthUser{}, ErrAuthUserNotFound
		}
		return accessdomain.AuthUser{}, fmt.Errorf("get auth user: %w", err)
	}
	if passwordHash == "" || !verifyPassword(password, passwordHash) {
		return accessdomain.AuthUser{}, fmt.Errorf("invalid credentials")
	}
	return user, nil
}

func (s *Service) AuthenticateTelegram(ctx context.Context, telegramID int64, telegramUsername string) (accessdomain.AuthUser, error) {
	if telegramID > 0 {
		account, err := s.repo.GetByTelegramID(ctx, telegramID)
		if err == nil {
			return account, nil
		}
		if !errors.Is(err, ErrAuthUserNotFound) {
			return accessdomain.AuthUser{}, err
		}
	}

	account, err := s.repo.GetByTelegramUsername(ctx, strings.TrimSpace(telegramUsername))
	if err != nil {
		return accessdomain.AuthUser{}, err
	}
	return s.repo.BindTelegramID(ctx, account.ID, telegramID)
}

func (s *Service) BindTelegram(ctx context.Context, userID, telegramID int64) (accessdomain.AuthUser, error) {
	if telegramID <= 0 {
		return accessdomain.AuthUser{}, fmt.Errorf("telegram id is required")
	}
	return s.repo.BindTelegramID(ctx, userID, telegramID)
}

func (s *Service) SetPassword(ctx context.Context, id int64, password string) (accessdomain.AuthUser, error) {
	if password == "" {
		return accessdomain.AuthUser{}, fmt.Errorf("password is required")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return accessdomain.AuthUser{}, err
	}
	return s.repo.SetPasswordHash(ctx, id, hash)
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) AssignUserDepartment(ctx context.Context, userID int64, departmentID *int64) (accessdomain.AuthUser, error) {
	return s.repo.AssignUserDepartment(ctx, userID, departmentID)
}

func (s *Service) GetUserByID(ctx context.Context, id int64) (accessdomain.AuthUser, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetUserByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error) {
	return s.repo.GetByTelegramID(ctx, telegramID)
}

func (s *Service) ListUsers(ctx context.Context) ([]accessdomain.AuthUser, error) {
	return s.repo.ListAll(ctx)
}

func (s *Service) SetUserRole(ctx context.Context, id int64, role string) (accessdomain.AuthUser, error) {
	role = accessdomain.NormalizeRole(role)
	if !accessdomain.IsValidRole(role) {
		return accessdomain.AuthUser{}, fmt.Errorf("%w: %s", ErrInvalidRole, role)
	}
	return s.repo.SetRole(ctx, id, role)
}

func (s *Service) SetUsername(ctx context.Context, id int64, username string) (accessdomain.AuthUser, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return accessdomain.AuthUser{}, apperr.Invalid("auth.username_required", "Укажите логин.")
	}
	return s.repo.SetUsername(ctx, id, username)
}

func (s *Service) GetUserByTelegramUsername(ctx context.Context, telegramUsername string) (accessdomain.AuthUser, error) {
	return s.repo.GetByTelegramUsername(ctx, strings.TrimSpace(telegramUsername))
}

func (s *Service) ListUsersByDepartmentID(ctx context.Context, departmentID int64) ([]accessdomain.AuthUser, error) {
	return s.repo.ListByDepartmentID(ctx, departmentID)
}

func (s *Service) ListUsersByRole(ctx context.Context, role string) ([]accessdomain.AuthUser, error) {
	return s.repo.ListByRole(ctx, accessdomain.NormalizeRole(role))
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

func verifyPassword(password, encoded string) bool {
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
