package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. Mod or admin runs `/campaigndatabase`.
		2. Authorize: check if the invoker has ScopeMod (mod or admin). Reject if not.
		3. Load all campaigns from the DB via `db.GetAll`.
		4. Build a list showing each campaign's name, tag, DM, and status flags (approved/unapproved, archived, open/closed).
		5. Respond ephemerally with the full list (truncated at 1900 chars if needed).
*/

/*
campaignDatabaseCommand shows all campaigns in the DB (approved, unapproved, archived).

This command is Staff only.
*/
type campaignDatabaseCommand struct {
	db *bun.DB
}

func (c *campaignDatabaseCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.CampaignDBCommandName,
		Description: messages.CampaignDBCommandDesc,
	}
}

func (c *campaignDatabaseCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	ok, err := auth.Authorize(c.db, userID, auth.ScopeMod, "")
	if err != nil || !ok {
		respond(s, i, messages.CampaignDBNotStaff)
		return
	}

	campaigns, err := db.GetAll[models.Campaign](c.db)
	if err != nil {
		log.Printf("campaign_database: failed to load campaigns: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	if len(campaigns) == 0 {
		respond(s, i, messages.CampaignDBEmpty)
		return
	}

	var lines []string
	for _, camp := range campaigns {
		flags := buildFlags(camp)
		lines = append(lines, fmt.Sprintf("**%s** (`%s`) — DM: <@%s> [%s]", camp.Name, camp.Tag, camp.DungeonMaster, flags))
	}

	content := "**All campaigns in database:**\n" + strings.Join(lines, "\n")

	if len(content) > 1900 {
		content = content[:1900] + "\n... (truncated)"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func buildFlags(c models.Campaign) string {
	var flags []string
	if c.IsApproved {
		flags = append(flags, "approved")
	} else {
		flags = append(flags, "unapproved")
	}
	if c.IsArchived {
		flags = append(flags, "archived")
	}
	if c.IsOpen {
		flags = append(flags, "open")
	} else {
		flags = append(flags, "closed")
	}
	return strings.Join(flags, ", ")
}
