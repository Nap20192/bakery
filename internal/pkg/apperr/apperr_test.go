package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestAsExtractsKindThroughWrapping(t *testing.T) {
	sentinel := NotFound("x.not_found", "missing")
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "direct", err: sentinel, want: KindNotFound},
		{name: "wrapped fmt", err: fmt.Errorf("load: %w", sentinel), want: KindNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := As(tt.err)
			if ae == nil || ae.Kind != tt.want {
				t.Fatalf("As() = %v, want kind %v", ae, tt.want)
			}
		})
	}
	if As(errors.New("plain")) != nil {
		t.Fatal("As must return nil for non-apperr errors")
	}
	if As(nil) != nil {
		t.Fatal("As must return nil for nil")
	}
}

func TestSentinelIdentity(t *testing.T) {
	sentinel := Conflict("x.dup", "duplicate")
	wrapped := fmt.Errorf("create: %w", sentinel)
	if !errors.Is(wrapped, sentinel) {
		t.Fatal("errors.Is must match the wrapped sentinel by identity")
	}
	if ae := As(wrapped); ae == nil || ae.Message != "duplicate" {
		t.Fatalf("As(wrapped) = %v, want message %q", ae, "duplicate")
	}
}

func TestErrorStringIncludesCode(t *testing.T) {
	if got := Invalid("x.bad", "плохие данные").Error(); got != "x.bad: плохие данные" {
		t.Fatalf("Error() = %q", got)
	}
	if got := (&Error{Message: "только текст"}).Error(); got != "только текст" {
		t.Fatalf("Error() without code = %q", got)
	}
}
