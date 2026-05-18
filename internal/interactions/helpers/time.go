package helpers

import (
	"fmt"
	"time"
)

/*
	DateTime helpers
*/

// TimeRemaining returns a human-readable "in X days/hours/minutes" string relative to now.
func TimeRemaining(t time.Time) string {
	d := time.Until(t).Round(time.Minute)
	if d <= 0 {
		return "now"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h >= 24*14:
		return fmt.Sprintf("in %d weeks", h/168)
	case h >= 24:
		days := h / 24
		if h%24 == 0 {
			return fmt.Sprintf("in %d days", days)
		}
		return fmt.Sprintf("in %d days %dh", days, h%24)
	case h > 0:
		if m == 0 {
			return fmt.Sprintf("in %dh", h)
		}
		return fmt.Sprintf("in %dh %dm", h, m)
	default:
		return fmt.Sprintf("in %dm", m)
	}
}

/*
FormatInLocation formats t in the given location using the provided layout.

Falls back to UTC if loc is nil.
*/
func FormatInLocation(t time.Time, layout string, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format(layout)
}

// TZLabel returns a short label like "CLST (UTC-4)" for display in modal fields.
func TZLabel(loc *time.Location) string {
	if loc == nil || loc == time.UTC {
		return "UTC"
	}
	now := time.Now().In(loc)
	abbr, offset := now.Zone()
	h := offset / 3600
	sign := "+"
	if h < 0 {
		sign = "-"
		h = -h
	}
	return fmt.Sprintf("%s (UTC%s%d)", abbr, sign, h)
}
