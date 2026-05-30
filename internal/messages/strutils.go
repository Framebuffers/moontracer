package messages

import (
	"strings"

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
	flags := []string{"unapproved"}
	if c.IsApproved {
		flags[0] = "approved"
	}
	if c.IsArchived {
		flags = append(flags, "archived", "closed")
	} else if c.IsOpen {
		flags = append(flags, "open")
	} else {
		flags = append(flags, "closed")
	}
	return strings.Join(flags, ", ")
}
