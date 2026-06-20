package messages

import (
	"regexp"

	"github.com/framebuffers/moontracer/internal/manager/models"
)

// snowflakeRe matches Discord snowflake IDs: 17–20 decimal digits.
var snowflakeRe = regexp.MustCompile(`^[0-9]{17,20}$`)

/*
BuildFlags renders a Campaign's status flags (approved/unapproved, archived,
open/closed) as a comma-separated string for display.
*/
func BuildFlags(c models.Campaign) string {
	if !c.DeletedAt.IsZero() {
		return "deleted"
	}
	if c.IsArchived {
		return "archived"
	}
	if !c.IsApproved {
		return "pending"
	}
	if c.IsOpen {
		return "active, open"
	}
	return "active, closed"
}

// IsSnowflake reports whether s looks like a Discord snowflake ID (17–20 decimal digits).
func IsSnowflake(s string) bool {
	return snowflakeRe.MatchString(s)
}
