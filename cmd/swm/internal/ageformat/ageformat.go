// Package ageformat formats a time.Time as a human-readable age string.
package ageformat

import (
	"fmt"
	"math"
	"time"
)

const (
	hoursInDay   = 24.0
	hoursInWeek  = 7 * hoursInDay
	hoursInMonth = 4 * hoursInWeek  // threshold: 4 weeks = 28 days
	hoursInYear  = 365 * hoursInDay // threshold: 365 days
)

// FormatAge returns a rounded-up single-unit age string for t relative to now.
// The unit is chosen by the raw duration: <1h→minutes, <24h→hours, <7d→days,
// <28d→weeks, <365d→months, ≥365d→years. The count within the unit is always
// rounded up so the caller never underestimates how old something is.
func FormatAge(t, now time.Time) string {
	d := now.Sub(t)
	if d <= 0 {
		return "1m ago"
	}

	h := d.Hours()

	switch {
	case h < 1:
		m := max(int(math.Ceil(d.Minutes())), 1)

		return fmt.Sprintf("%dm ago", m)

	case h < hoursInDay:
		return fmt.Sprintf("%dh ago", int(math.Ceil(h)))

	case h < hoursInWeek:
		return fmt.Sprintf("%dd ago", int(math.Ceil(h/hoursInDay)))

	case h < hoursInMonth:
		return fmt.Sprintf("%dw ago", int(math.Ceil(h/hoursInWeek)))

	case h < hoursInYear:
		// Use 30.44 days/month for display so short months don't round oddly.
		return fmt.Sprintf("%dmo ago", int(math.Ceil(h/(30.44*hoursInDay))))

	default:
		return fmt.Sprintf("%dy ago", int(math.Ceil(h/hoursInYear)))
	}
}
