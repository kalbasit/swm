package ageformat_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/ageformat"
)

const oneMinAgo = "1m ago"

func TestFormatAge(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		// Sub-hour: rounds up to minutes
		{"zero age", 0, oneMinAgo},
		{"negative age", -time.Minute, oneMinAgo},
		{"exactly 1 minute", time.Minute, oneMinAgo},
		{"47m 30s rounds up to 48m", 47*time.Minute + 30*time.Second, "48m ago"},
		{"59m 59s rounds up to 60m", 59*time.Minute + 59*time.Second, "60m ago"},

		// Sub-day: rounds up to hours (boundary: 1h to <24h)
		{"exactly 1 hour", time.Hour, "1h ago"},
		{"23h 1min rounds up to 24h", 23*time.Hour + time.Minute, "24h ago"},
		{"2h exactly", 2 * time.Hour, "2h ago"},
		{"1h 30min rounds up to 2h", time.Hour + 30*time.Minute, "2h ago"},

		// Sub-week: rounds up to days (boundary: 24h to <168h)
		{"exactly 1 day", 24 * time.Hour, "1d ago"},
		{"6d 2h rounds up to 7d", 6*24*time.Hour + 2*time.Hour, "7d ago"},
		{"2 days exactly", 2 * 24 * time.Hour, "2d ago"},

		// Sub-month: rounds up to weeks (boundary: 7 days to <28 days)
		{"exactly 7 days", 7 * 24 * time.Hour, "1w ago"},
		{"13 days exactly", 13 * 24 * time.Hour, "2w ago"},
		{"14 days", 14 * 24 * time.Hour, "2w ago"},
		{"27 days 23h rounds up to 4w", 27*24*time.Hour + 23*time.Hour, "4w ago"},

		// Sub-year: rounds up to months (boundary: 28 days to <365 days)
		{"exactly 28 days", 28 * 24 * time.Hour, "1mo ago"},
		{"60 days", 60 * 24 * time.Hour, "2mo ago"},
		{"364 days", 364 * 24 * time.Hour, "12mo ago"},

		// Years (boundary: ≥365 days)
		{"exactly 365 days", 365 * 24 * time.Hour, "1y ago"},
		{"730 days", 730 * 24 * time.Hour, "2y ago"},
		{"2 years 1 day", 731 * 24 * time.Hour, "3y ago"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ageformat.FormatAge(now.Add(-tc.age), now)
			require.Equal(t, tc.want, got)
		})
	}
}
