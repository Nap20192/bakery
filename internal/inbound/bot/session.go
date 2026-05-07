package bot

import (
	"log/slog"
	"time"

	orderdomain "bakery/internal/domain/order"
)

const (
	sessionTTL      = 12 * time.Hour // незавершённая заявка живёт 12 часов
	cleanupInterval = time.Hour      // проверяем каждый час
)

type session struct {
	items     []orderdomain.OrderItem
	updatedAt time.Time // время последнего изменения
}

// updateSession атомарно изменяет сессию пользователя под мьютексом.
func (b *OrderBot) updateSession(uid int64, fn func(*session)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, ok := b.sessions[uid]
	if !ok {
		s = &session{}
		b.sessions[uid] = s
	}
	fn(s)
	s.updatedAt = time.Now()
}

func (b *OrderBot) clearSession(uid int64) {
	b.updateSession(uid, func(s *session) {
		s.items = nil
	})
}

// cleanupLoop удаляет сессии, которые не обновлялись дольше sessionTTL.
func (b *OrderBot) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		b.mu.Lock()
		now := time.Now()
		var expired []int64
		for uid, s := range b.sessions {
			if !s.updatedAt.IsZero() && now.Sub(s.updatedAt) > sessionTTL {
				expired = append(expired, uid)
			}
		}
		for _, uid := range expired {
			delete(b.sessions, uid)
		}
		b.mu.Unlock()
		if len(expired) > 0 {
			slog.Info("session cleanup", "expired", len(expired))
		}
	}
}
