package httpx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestValidateMiniAppInitData(t *testing.T) {
	now := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	const botToken = "123456:bot-token"

	tests := []struct {
		name    string
		build   func() string
		wantID  int64
		wantErr bool
	}{
		{
			name: "valid data",
			build: func() string {
				return signedInitData(botToken, now.Unix(), `{"id":711,"username":"shop"}`)
			},
			wantID: 711,
		},
		{
			name: "tampered user",
			build: func() string {
				values, _ := url.ParseQuery(signedInitData(botToken, now.Unix(), `{"id":711}`))
				values.Set("user", `{"id":999}`)
				return values.Encode()
			},
			wantErr: true,
		},
		{
			name: "expired data",
			build: func() string {
				return signedInitData(botToken, now.Add(-miniAppAuthMaxAge-time.Second).Unix(), `{"id":711}`)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := validateMiniAppInitData(tt.build(), botToken, now, miniAppAuthMaxAge)
			if tt.wantErr {
				if err == nil {
					t.Fatal("validateMiniAppInitData() returned nil error")
				}
				return
			}
			if err != nil {
				t.Fatalf("validateMiniAppInitData() error = %v", err)
			}
			if user.ID != tt.wantID {
				t.Fatalf("user.ID = %d, want %d", user.ID, tt.wantID)
			}
		})
	}
}

func TestAuthorizationCredentials(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantScheme string
		wantData   string
		ok         bool
	}{
		{name: "tma scheme", header: "tma auth_date=1&hash=a", wantScheme: "tma", wantData: "auth_date=1&hash=a", ok: true},
		{name: "bearer scheme", header: "Bearer v1.body.sig", wantScheme: "Bearer", wantData: "v1.body.sig", ok: true},
		{name: "empty data", header: "tma ", ok: false},
		{name: "no scheme", header: "garbage", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, data, ok := authorizationCredentials(tt.header)
			if ok != tt.ok || (ok && (scheme != tt.wantScheme || data != tt.wantData)) {
				t.Fatalf("authorizationCredentials() = (%q, %q, %v), want (%q, %q, %v)", scheme, data, ok, tt.wantScheme, tt.wantData, tt.ok)
			}
		})
	}
}

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
