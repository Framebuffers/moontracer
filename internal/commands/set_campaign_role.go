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
		1. Extract the invoker ID, the campaign to assign the role from, and the name of the role.
		2. Authorize
		3. Fetch roles from guild.
		4. Get the role that matches the Campaign. If there isn't one, create it.
		5. Update Campaign with new role ID.
		6. Update DB.
*/

type setCampaignRole struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (r *setCampaignRole) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.SetCampaignRoleCommandName,
		Description: messages.SetCampaignRoleCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        messages.TagCommandName,
				Description: messages.TagCommandDesc,
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        messages.SetRoleFieldName,
				Description: messages.SetRoleFieldDesc,
				Required:    true,
			},
		},
	}
}

// Execute is the logic that runs when the user invokes the command.
func (r *setCampaignRole) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := i.Member.User.ID
	tag := i.ApplicationCommandData().Options[0].StringValue()
	roleName := i.ApplicationCommandData().Options[1].StringValue()

	campaign, err := db.GetByTag[models.Campaign](r.db, tag)
	if err != nil {
		respond(s, i, messages.CampaignNotFoundMessage)
		return
	}

	// Only the DM can manage campaign settings — campaign sovereignty.
	ok, err := auth.Authorize(r.db, invokerID, auth.ScopeDM, campaign.ID)
	if err != nil {
		log.Printf("set_campaign_role: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.SetRoleNotDMOrMod)
		return
	}

	var roleID string

	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		log.Printf("set_campaign_role: failed to fetch roles from guild: %v", err)
		respond(s, i, "Failed to get roles from guild.")
		return
	}

	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			roleID = role.ID
			break
		}
	}

	if roleID == "" {
		log.Printf("set_campaign_role: no role found, creating new one.")
		role, err := s.GuildRoleCreate(i.GuildID, &discordgo.RoleParams{
			Name: roleName,
		})
		if err != nil {
			log.Printf("set_campaign_role: failed to create role: %v", err)
			respond(s, i, messages.SetRoleCreateFailed)
			return
		}
		roleID = role.ID
	}

	campaign.RoleID = roleID
	if err := db.Update(r.db, campaign); err != nil {
		log.Printf("set_campaign_role: failed to update campaign: %v", err)
		respond(s, i, messages.SetRoleUpdateFailed)
		return
	}

	// Assign the campaign role to the DM.
	if err := s.GuildMemberRoleAdd(i.GuildID, invokerID, roleID); err != nil {
		log.Printf("set_campaign_role: failed to add role %s to member %s: %v", roleID, invokerID, err)
		respond(s, i, messages.SetRoleUpdateFailed)
		return
	}

	respond(s, i, fmt.Sprintf(messages.SetRoleSuccess, roleName, campaign.Name))
}
