package bot

import (
	"errors"
	"fmt"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestValidateMiniAppURL(t *testing.T) {
	for raw, wantErr := range map[string]bool{
		"":                       false,
		"https://bakery.example": false,
		"http://localhost:8080":  true,
		"bakery.example":         true,
		"https://":               true,
	} {
		if err := ValidateMiniAppURL(raw); (err != nil) != wantErr {
			t.Errorf("ValidateMiniAppURL(%q) error = %v, want error %v", raw, err, wantErr)
		}
	}
}

func TestIsRejectedToken(t *testing.T) {
	if !IsRejectedToken(fmt.Errorf("orderbot: new: %w", tele.ErrUnauthorized)) {
		t.Error("401 must be treated as a rejected token")
	}
	if !IsRejectedToken(fmt.Errorf("orderbot: new: %w", tele.ErrNotFound)) {
		t.Error("404 must be treated as a rejected token")
	}
	if IsRejectedToken(errors.New("dial tcp: i/o timeout")) {
		t.Error("network errors must be retried")
	}
	if IsRejectedToken(fmt.Errorf("orderbot: new: %w", tele.ErrInternal)) {
		t.Error("5xx must be retried")
	}
}
