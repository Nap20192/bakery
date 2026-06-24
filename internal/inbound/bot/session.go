package bot

import (
	"log/slog"
	"time"
)

const (
	sessionTTL      = 1 * time.Hour // ожидание ввода пароля живёт час
	cleanupInterval = time.Hour     // проверяем каждый час
)

// session tracks the /start password gate: when awaitingPassword is set the
// next text message from the user is treated as the password for `username`.
type session struct {
	awaitingPassword bool
	username         string
	updatedAt        time.Time
}

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

func (b *OrderBot) getSession(uid int64) session {
	b.mu.Lock()
	defer b.mu.Unlock()
	if s, ok := b.sessions[uid]; ok {
		return *s
	}
	return session{}
}

func (b *OrderBot) resetSession(uid int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.sessions, uid)
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
