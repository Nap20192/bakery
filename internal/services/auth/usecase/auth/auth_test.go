package authuc

import (
	"context"
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

type fakeAuthRepo struct {
	Repository
	byTelegramUsername map[string]accessdomain.AuthUser
	boundUserID        int64
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

func strPtr(value string) *string {
	return &value
}
