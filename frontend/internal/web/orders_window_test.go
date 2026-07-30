package web

import (
	"testing"
	"time"
)

func TestNormalizeMatrixWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 29, 15, 30, 0, 0, time.UTC)
	tests := []struct {
		name     string
		from     string
		to       string
		wantFrom string
		wantTo   string
	}{
		{
			name:     "keeps documented five day window",
			from:     "2026-08-03",
			to:       "2026-08-07",
			wantFrom: "2026-08-03",
			wantTo:   "2026-08-07",
		},
		{
			name:     "defaults missing window",
			wantFrom: "2026-07-29",
			wantTo:   "2026-08-02",
		},
		{
			name:     "defaults reversed window",
			from:     "2026-08-07",
			to:       "2026-08-03",
			wantFrom: "2026-07-29",
			wantTo:   "2026-08-02",
		},
		{
			name:     "defaults unbounded window",
			from:     "0001-01-01",
			to:       "9999-12-31",
			wantFrom: "2026-07-29",
			wantTo:   "2026-08-02",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters := orderFiltersView{
				FulfillmentFrom: test.from,
				FulfillmentTo:   test.to,
			}
			normalizeMatrixWindow(&filters, now)
			if filters.FulfillmentFrom != test.wantFrom || filters.FulfillmentTo != test.wantTo {
				t.Fatalf("window = %s..%s, want %s..%s",
					filters.FulfillmentFrom,
					filters.FulfillmentTo,
					test.wantFrom,
					test.wantTo,
				)
			}
		})
	}
}
