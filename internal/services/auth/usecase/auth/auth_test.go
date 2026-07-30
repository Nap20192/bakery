package authuc

import (
	"context"
	"fmt"
	"testing"

	accessdomain "bakery/internal/services/auth/domain"
)

func TestServiceAuthenticateTelegramBindsByTelegramUsername(t *testing.T) {
	repo := &fakeAuthRepo{
		byTelegramUsername: map[string]accessdomain.AuthUser{
			"shop_user": {ID: 7, Username: "shop", TelegramUsername: strPtr("shop_user")},
		},
	}
	svc := NewService(repo)

	user, err := svc.AuthenticateTelegram(context.Background(), 123456789, "shop_user")
	if err != nil {
		t.Fatalf("AuthenticateTelegram returned error: %v", err)
	}
	if user.TelegramID == nil || *user.TelegramID != 123456789 {
		t.Fatalf("telegram id = %#v, want 123456789", user.TelegramID)
	}
	if repo.boundUserID != 7 {
		t.Fatalf("bound user id = %d, want 7", repo.boundUserID)
	}
}

func TestServiceEnsureAdminUserKeepsExistingAccount(t *testing.T) {
	existing := accessdomain.AuthUser{ID: 7, Username: "admin", Role: accessdomain.RoleAdmin}
	repo := &fakeAuthRepo{byUsername: existing}

	user, created, err := NewService(repo).EnsureAdminUser(context.Background(), " admin ", "new-password")
	if err != nil {
		t.Fatalf("EnsureAdminUser returned error: %v", err)
	}
	if created {
		t.Fatal("created = true, want false")
	}
	if user.ID != existing.ID {
		t.Fatalf("user id = %d, want %d", user.ID, existing.ID)
	}
	if repo.createCalls != 0 {
		t.Fatalf("CreatePasswordUser calls = %d, want 0", repo.createCalls)
	}
}

func TestServiceEnsureAdminUserHandlesConcurrentCreate(t *testing.T) {
	existing := accessdomain.AuthUser{ID: 9, Username: "admin", Role: accessdomain.RoleAdmin}
	repo := &fakeAuthRepo{
		byUsername:       existing,
		missUsernameOnce: true,
		createErr:        fmt.Errorf("concurrent create: %w", ErrUsernameTaken),
	}

	user, created, err := NewService(repo).EnsureAdminUser(context.Background(), "admin", "password")
	if err != nil {
		t.Fatalf("EnsureAdminUser returned error: %v", err)
	}
	if created {
		t.Fatal("created = true, want false")
	}
	if user.ID != existing.ID {
		t.Fatalf("user id = %d, want %d", user.ID, existing.ID)
	}
	if repo.getByUsernameCalls != 2 {
		t.Fatalf("GetByUsername calls = %d, want 2", repo.getByUsernameCalls)
	}
	if repo.createCalls != 1 {
		t.Fatalf("CreatePasswordUser calls = %d, want 1", repo.createCalls)
	}
}

type fakeAuthRepo struct {
	Repository
	byTelegramUsername map[string]accessdomain.AuthUser
	boundUserID        int64
	byUsername         accessdomain.AuthUser
	missUsernameOnce   bool
	getByUsernameCalls int
	createErr          error
	createCalls        int
}

func (r *fakeAuthRepo) GetByTelegramUsername(_ context.Context, telegramUsername string) (accessdomain.AuthUser, error) {
	user, ok := r.byTelegramUsername[telegramUsername]
	if !ok {
		return accessdomain.AuthUser{}, ErrAuthUserNotFound
	}
	return user, nil
}

func (r *fakeAuthRepo) BindTelegramID(_ context.Context, id, telegramID int64) (accessdomain.AuthUser, error) {
	r.boundUserID = id
	user := accessdomain.AuthUser{ID: id, TelegramID: &telegramID}
	return user, nil
}

func (r *fakeAuthRepo) GetByUsername(_ context.Context, _ string) (accessdomain.AuthUser, string, error) {
	r.getByUsernameCalls++
	if r.missUsernameOnce && r.getByUsernameCalls == 1 {
		return accessdomain.AuthUser{}, "", ErrAuthUserNotFound
	}
	if r.byUsername.ID == 0 {
		return accessdomain.AuthUser{}, "", ErrAuthUserNotFound
	}
	return r.byUsername, "stored-hash", nil
}

func (r *fakeAuthRepo) CreatePasswordUser(_ context.Context, _ CreatePasswordUserInput) (accessdomain.AuthUser, error) {
	r.createCalls++
	return accessdomain.AuthUser{}, r.createErr
}

func strPtr(value string) *string {
	return &value
}
