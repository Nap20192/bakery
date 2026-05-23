package bot

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseTestOrderDate(t *testing.T) {
	now := time.Date(2026, time.May, 21, 12, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		raw  string
		want time.Time
	}{
		{
			name: "future day in current month",
			raw:  "25",
			want: time.Date(2026, time.May, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "today",
			raw:  "21",
			want: time.Date(2026, time.May, 21, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "past day moves to next month",
			raw:  "20",
			want: time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "skips month without day",
			raw:  "31",
			want: time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTestOrderDate(now, tt.raw)
			if err != nil {
				t.Fatalf("parseTestOrderDate() error = %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("parseTestOrderDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTestOrderDateInvalid(t *testing.T) {
	now := time.Date(2026, time.May, 21, 12, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty", raw: ""},
		{name: "text", raw: "abc"},
		{name: "zero", raw: "0"},
		{name: "too large", raw: "32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTestOrderDate(now, tt.raw)
			if !errors.Is(err, errInvalidTestOrderDay) {
				t.Fatalf("parseTestOrderDate() error = %v, want %v", err, errInvalidTestOrderDay)
			}
		})
	}
}

func TestTestOrderDateErrorMessage(t *testing.T) {
	got := testOrderDateErrorMessage(errInvalidTestOrderDay)
	want := "Укажите число месяца от 1 до 31."
	if got != want {
		t.Fatalf("testOrderDateErrorMessage() = %q, want %q", got, want)
	}
}

func TestTestOrdersResultMessage(t *testing.T) {
	date := time.Date(2026, time.May, 25, 0, 0, 0, 0, time.UTC)
	msg := testOrdersResultMessage(date, []string{"Магазин Гагарина: Г.25.05.26.001"}, []string{"Магазин Шолохова: ошибка создания"})

	for _, want := range []string{
		"Тестовые заказы на 25.05.2026",
		"Созданы:",
		"Магазин Гагарина: Г.25.05.26.001",
		"Не созданы:",
		"Магазин Шолохова: ошибка создания",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("testOrdersResultMessage() = %q, want to contain %q", msg, want)
		}
	}
}
