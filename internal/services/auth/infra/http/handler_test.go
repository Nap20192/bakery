package authhttp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	accessdomain "bakery/internal/services/auth/domain"
	authuc "bakery/internal/services/auth/usecase/auth"

	"context"
)

const testBotToken = "123456:bot-token"

type fakeAuth struct {
	authuc.UseCase
	boundUserID     int64
	boundTelegramID int64
}

func (f *fakeAuth) VerifyPassword(_ context.Context, username, password string) (accessdomain.AuthUser, error) {
	return accessdomain.AuthUser{ID: 42, Username: username}, nil
}

func (f *fakeAuth) BindTelegram(_ context.Context, userID, telegramID int64) (accessdomain.AuthUser, error) {
	f.boundUserID = userID
	f.boundTelegramID = telegramID
	return accessdomain.AuthUser{ID: userID}, nil
}

func postLogin(t *testing.T, svc *fakeAuth, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := New(svc, nil, testBotToken)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.handleLogin(rec, req)
	return rec
}

func TestLoginBindsTelegramFromInitData(t *testing.T) {
	svc := &fakeAuth{}
	init := signedInitData(testBotToken, time.Now().Unix(), `{"id":711,"username":"Shop_User"}`)
	rec := postLogin(t, svc, `{"username":"shop","password":"pw","init_data":`+strconv.Quote(init)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body %s", rec.Code, rec.Body.String())
	}
	if svc.boundUserID != 42 || svc.boundTelegramID != 711 {
		t.Fatalf("bind = (user %d, telegram %d), want (42, 711)", svc.boundUserID, svc.boundTelegramID)
	}
}

func TestLoginIgnoresInvalidInitData(t *testing.T) {
	svc := &fakeAuth{}
	rec := postLogin(t, svc, `{"username":"shop","password":"pw","init_data":"auth_date=1&hash=00"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body %s", rec.Code, rec.Body.String())
	}
	if svc.boundTelegramID != 0 {
		t.Fatalf("bind happened with invalid init data (telegram %d)", svc.boundTelegramID)
	}
}

// signedInitData mirrors Telegram's initData signing (same as the httpx tests).
func signedInitData(botToken string, authDate int64, user string) string {
	values := url.Values{
		"auth_date": {strconv.FormatInt(authDate, 10)},
		"query_id":  {"query"},
		"user":      {user},
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+values.Get(key))
	}
	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMAC.Write([]byte(botToken))
	signatureMAC := hmac.New(sha256.New, secretMAC.Sum(nil))
	_, _ = signatureMAC.Write([]byte(strings.Join(lines, "\n")))
	values.Set("hash", hex.EncodeToString(signatureMAC.Sum(nil)))
	return values.Encode()
}
