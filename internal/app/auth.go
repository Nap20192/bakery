package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"bakery/internal/domain/client"
	"bakery/internal/domain/kitchen"
	"bakery/internal/domain/order"
	"bakery/internal/domain/user"
)

var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrForbidden          = errors.New("auth: forbidden")
	ErrInvalidToken       = errors.New("auth: invalid token")
	ErrLoginTaken         = errors.New("auth: login already taken")
)

// Claims — то, что подписывается и кладётся в контекст запроса.
type Claims struct {
	UserID uuid.UUID `json:"sub"`
	Role   user.Role `json:"role"`
	Exp    int64     `json:"exp"`
}

// AuthService — прикладной сервис.
// Использует внешние зависимости (bcrypt, HMAC-JWT) напрямую,
// а с доменом работает только через репозитории-порты.
type AuthService struct {
	users    user.Repository
	clients  client.Repository
	kitchens kitchen.Repository
	secret   []byte
	ttl      time.Duration
}

func NewAuthService(
	users user.Repository,
	clients client.Repository,
	kitchens kitchen.Repository,
	secret string,
	ttl time.Duration,
) *AuthService {
	return &AuthService{
		users:    users,
		clients:  clients,
		kitchens: kitchens,
		secret:   []byte(secret),
		ttl:      ttl,
	}
}

// RegisterClient создаёт user + client в один прикладной вызов.
// Доменные инварианты проверяет домен (user.New, client.New).
func (s *AuthService) RegisterClient(ctx context.Context, login, password, name string) (*user.User, *client.Client, error) {
	u, err := s.createUser(ctx, login, password, user.RoleClient)
	if err != nil {
		return nil, nil, err
	}
	uid := u.ID()
	c, err := client.New(name, &uid, nil, "")
	if err != nil {
		return nil, nil, fmt.Errorf("auth: build client: %w", err)
	}
	if err := s.clients.Save(ctx, c); err != nil {
		return nil, nil, fmt.Errorf("auth: save client: %w", err)
	}
	return u, c, nil
}

// RegisterKitchen создаёт user + kitchen.
func (s *AuthService) RegisterKitchen(ctx context.Context, login, password, name string) (*user.User, *kitchen.Kitchen, error) {
	u, err := s.createUser(ctx, login, password, user.RoleKitchen)
	if err != nil {
		return nil, nil, err
	}
	uid := u.ID()
	k, err := kitchen.New(name, &uid)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: build kitchen: %w", err)
	}
	if err := s.kitchens.Save(ctx, k); err != nil {
		return nil, nil, fmt.Errorf("auth: save kitchen: %w", err)
	}
	return u, k, nil
}

func (s *AuthService) createUser(ctx context.Context, login, password string, role user.Role) (*user.User, error) {
	if _, err := s.users.GetByLogin(ctx, login); err == nil {
		return nil, ErrLoginTaken
	} else if !errors.Is(err, user.ErrNotFound) {
		return nil, fmt.Errorf("auth: lookup login: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}
	u, err := user.New(login, string(hash), role)
	if err != nil {
		return nil, fmt.Errorf("auth: build user: %w", err)
	}
	if err := s.users.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("auth: save user: %w", err)
	}
	return u, nil
}

// Login — проверка пароля и выдача токена.
func (s *AuthService) Login(ctx context.Context, login, password string) (string, Claims, error) {
	u, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		return "", Claims{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash()), []byte(password)); err != nil {
		return "", Claims{}, ErrInvalidCredentials
	}
	claims := Claims{
		UserID: u.ID(),
		Role:   u.Role(),
		Exp:    time.Now().Add(s.ttl).Unix(),
	}
	token, err := s.sign(claims)
	if err != nil {
		return "", Claims{}, fmt.Errorf("auth: sign token: %w", err)
	}
	return token, claims, nil
}

// Verify — парсинг и проверка токена.
func (s *AuthService) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Claims{}, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(parts[0]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return Claims{}, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if time.Now().Unix() > c.Exp {
		return Claims{}, ErrInvalidToken
	}
	return c, nil
}

func (s *AuthService) sign(c Claims) (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	head := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(head))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return head + "." + sig, nil
}

// --- Authorization policies ---
// Политики живут в app-слое: они комбинируют несколько агрегатов (user + client/kitchen + order)
// и поэтому не относятся ни к одному доменному модулю.

func (s *AuthService) CanViewOrder(ctx context.Context, c Claims, o *order.Order) error {
	switch c.Role {
	case user.RoleAdmin, user.RoleKitchen:
		return nil
	case user.RoleClient:
		cl, err := s.clients.GetByUserID(ctx, c.UserID)
		if err != nil {
			return ErrForbidden
		}
		if cl.ID() != o.ClientID() {
			return ErrForbidden
		}
		return nil
	}
	return ErrForbidden
}

func (s *AuthService) CanCreateOrderAs(ctx context.Context, c Claims, asClientID client.ID) error {
	if c.Role == user.RoleAdmin {
		return nil
	}
	if c.Role != user.RoleClient {
		return ErrForbidden
	}
	cl, err := s.clients.GetByUserID(ctx, c.UserID)
	if err != nil || cl.ID() != asClientID {
		return ErrForbidden
	}
	return nil
}

func (s *AuthService) CanChangeOrderStatus(c Claims) error {
	if c.Role == user.RoleAdmin || c.Role == user.RoleKitchen {
		return nil
	}
	return ErrForbidden
}

func (s *AuthService) CanManageCatalog(c Claims) error {
	if c.Role == user.RoleAdmin {
		return nil
	}
	return ErrForbidden
}
