package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. A mod/admin invokes `/unban @player`
		2. Auth: invoker must be at least mod.
		3. Load target player, check they are actually banned.
		4. Clear IsBanned + BanReason.
		5. Handle campaign-status reversal (see TODO below).
		6. Respond with success or failure.
*/

type unbanCommand struct {
	db *bun.DB
}

func (u *unbanCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.UnbanCommandName,
		Description: messages.UnbanCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "player",
				Description: "The player to unban.",
				Required:    true,
			},
		},
	}
}

func (u *unbanCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := i.Member.User.ID
	targetUser := i.ApplicationCommandData().Options[0].UserValue(s)

	ok, err := auth.Authorize(u.db, invokerID, auth.ScopeMod, "")
	if err != nil {
		log.Printf("unban: invoker auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.AddPlayerNotDMOrModMessage)
		return
	}

	target, err := db.GetByID[models.Player](u.db, targetUser.ID)
	if err != nil {
		respond(s, i, messages.UnbanTargetNotFound)
		return
	}

	if !target.PlayerIsBanned {
		respond(s, i, messages.UnbanTargetNotBanned)
		return
	}

	target.PlayerIsBanned = false
	target.PlayerBanReason = ""

	if err := db.Update(u.db, target); err != nil {
		log.Printf("unban: failed to update player: %v", err)
		respond(s, i, messages.UnbanFailureMessage)
		return
	}

	// Cascade: restore campaign memberships banned by the global ban.
	// Campaign-scoped bans (BannedFromCampaign == true) are left untouched.
	campaigns, err := models.GetPlayerCampaigns(u.db, target.ID)
	if err != nil {
		log.Printf("unban: failed to load campaigns for %s: %v", target.ID, err)
	}

	skipCampaignBans := func(cp models.CampaignPlayer) bool {
		return cp.Status != models.StatusBanned || cp.BannedFromCampaign
	}

	restored, skipped, errs := models.BulkSetCampaignPlayerStatus(u.db, target.ID, campaigns, models.StatusActive, skipCampaignBans)
	if errs != nil {
		for campID, e := range errs {
			log.Printf("unban: failed to restore %s in campaign %s: %v", target.ID, campID, e)
		}
	}

	auditReason := fmt.Sprintf("restored %d campaign(s), %d campaign-scoped ban(s) preserved", restored, skipped)
	if err := models.InsertAuditEntry(u.db, target.ID, invokerID, models.AuditUnban, auditReason); err != nil {
		log.Printf("unban: failed to write audit entry: %v", err)
	}

	log.Printf("unban: %s unbanned %s (restored %d campaigns, %d campaign bans preserved)", invokerID, target.ID, restored, skipped)
	respond(s, i, fmt.Sprintf(messages.UnbanSuccessMessage, targetUser.ID))
}
