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
		1. Someone with permissions (a mod/admin) invokes `/ban @player [reason]`
		2. Check if you are trying to ban yourself, and that you are indeed authorized to ban someone.
			a. Calls `Authorize()`
			b. Checks that the invoker is, at least, a moderator.
		3. Loads both players, and checks their 'role weight':
			a. From less to more: unregistered -> member -> player -> DM -> mod -> admin
			b. Privilege is issued through a composition chain: access to things are 'added' like a hat on top of your current permissions.
			c. Putting a 'mod' hat on a player means that it *inherits* all the permissions granted from below.
			d. DM is special: it grants privileges *only* on Campaigns, and only in those you own. However, a mod can override this.
		4. Checks if the player is already banned. Bail out early if so.
		5. Set `Player.IsBanned = true` + the reason in the DB.
		6. Cascade through every single campaign this member is a player on, and set their state to `banned`.
		7. Respond with either success or failure.
*/

type banCommand struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (r *banCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.BanCommandName,
		Description: messages.BanCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "player",
				Description: "The player to ban.",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "Reason for the ban (optional).",
				Required:    false,
			},
		},
	}
}

// Execute is the logic that runs when the user invokes the ban command.
func (r *banCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := i.Member.User.ID
	targetUser := i.ApplicationCommandData().Options[0].UserValue(s)

	var reason string
	if len(i.ApplicationCommandData().Options) > 1 {
		reason = i.ApplicationCommandData().Options[1].StringValue()
	}

	// Can't ban yourself.
	if invokerID == targetUser.ID {
		respond(s, i, messages.BanCannotBanSelf)
		return
	}

	// Auth: invoker must be mod or admin.
	ok, err := auth.Authorize(r.db, invokerID, auth.ScopeMod, "")
	if err != nil {
		log.Printf("ban: invoker auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.AddPlayerNotDMOrModMessage)
		return
	}

	// Load both players to compare roles.
	invoker, err := db.GetByID[models.Player](r.db, invokerID)
	if err != nil {
		log.Printf("ban: failed to load invoker: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	target, err := db.GetByID[models.Player](r.db, targetUser.ID)
	if err != nil {
		respond(s, i, messages.BanTargetNotFound)
		return
	}

	// Role protection: invoker must outrank target.
	if invoker.Role.Weight() <= target.Role.Weight() {
		respond(s, i, messages.BanInsufficientRole)
		return
	}

	// Already banned?
	if target.IsBanned {
		respond(s, i, messages.BanTargetAlreadyBanned)
		return
	}

	// Persist the global ban.
	target.IsBanned = true
	target.BanReason = reason

	if err := db.Update(r.db, target); err != nil {
		log.Printf("ban: failed to update player: %v", err)
		respond(s, i, messages.BanFailureMessage)
		return
	}

	// Cascade: ban from all campaigns.
	campaigns, err := models.GetPlayerCampaigns(r.db, target.ID)
	if err != nil {
		log.Printf("ban: failed to load campaigns for %s: %v", target.ID, err)
		// Player is banned globally, but campaign cascade failed — still report success.
	}

	var failedCampaigns []string
	for _, cp := range campaigns {
		if err := models.SetCampaignPlayerStatus(r.db, target.ID, cp.CampaignID, models.StatusBanned); err != nil {
			log.Printf("ban: failed to ban %s from campaign %s: %v", target.ID, cp.CampaignID, err)
			failedCampaigns = append(failedCampaigns, cp.CampaignID)
		}
	}
	if len(failedCampaigns) > 0 {
		log.Printf("ban: failed to cascade ban to campaigns: %s", strings.Join(failedCampaigns, ", "))
	}

	// Log and respond.
	log.Printf("ban: %s banned %s (reason: %s)", invokerID, target.ID, reason)
	if reason == "" {
		respond(s, i, fmt.Sprintf(messages.BanSuccessMessage, targetUser.ID))
	} else {
		respond(s, i, fmt.Sprintf(messages.BanSuccessReasonMessage, targetUser.ID, reason))
	}
}
