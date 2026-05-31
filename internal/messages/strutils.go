package messages

import (
	"github.com/framebuffers/moontracer/internal/manager/models"
)

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
