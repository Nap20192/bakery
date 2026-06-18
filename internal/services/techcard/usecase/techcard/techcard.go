package techcarduc

import (
	"context"
	"fmt"
	"strings"
	"time"

	techcarddomain "bakery/internal/services/techcard/domain"
)

type Service struct {
	repo Repository
}

var _ UseCase = (*Service)(nil)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByCode(ctx context.Context, code string, date time.Time) (techcarddomain.TechCard, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return techcarddomain.TechCard{}, fmt.Errorf("code is required")
	}
	if date.IsZero() {
		date = time.Now().UTC()
	}
	return s.repo.GetByCode(ctx, code, date)
}
