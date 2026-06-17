package authtoken

import (
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	const secret = "bot-token-secret"
	now := time.Unix(1_700_000_000, 0)

	token, err := Issue(secret, 4242, time.Hour, now)
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}
	claims, err := Parse(secret, token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if claims.UserID != 4242 {
		t.Fatalf("user id = %d, want 4242", claims.UserID)
	}
	if _, err := Parse(secret, token, now.Add(2*time.Hour)); err == nil {
		t.Fatal("expected expired token error")
	}
	if _, err := Parse("other-secret", token, now.Add(time.Minute)); err == nil {
		t.Fatal("expected signature mismatch error")
	}
}
