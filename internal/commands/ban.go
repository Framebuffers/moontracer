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

/*
	Flow:
		1. Guard check: cannot ban yourself.
		2. Load both players to compare roles.
		3. Guard check: to protect roles, invoker musy outrank target.
		4. Guard check: is the player already banned?
		5. Persist the global ban.
*/

// Execute is the logic that runs when the user invokes the ban command.
func (r *banCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := i.Member.User.ID
	targetUser := i.ApplicationCommandData().Options[0].UserValue(s)

	var reason string
	if len(i.ApplicationCommandData().Options) > 1 {
		reason = i.ApplicationCommandData().Options[1].StringValue()
	}

	if invokerID == targetUser.ID {
		respond(s, i, messages.BanCannotBanSelf)
		return
	}

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

	if invoker.Role.Weight() <= target.Role.Weight() {
		respond(s, i, messages.BanInsufficientRole)
		return
	}

	if target.PlayerIsBanned {
		respond(s, i, messages.BanTargetAlreadyBanned)
		return
	}

	target.PlayerIsBanned = true
	target.PlayerBanReason = reason

	if err := db.Update(r.db, target); err != nil {
		log.Printf("ban: failed to update player: %v", err)
		respond(s, i, messages.BanFailureMessage)
		return
	}

	/*

		A fundamental concept for Moontracer is: DM Sovereignty.
		The **only** true sovereign of a Campaign is the one that created, managed and played it.
		Therefore, permissions work a bit differently than *just* a regular hierarchical structure.

		The global ban flag blocks all auth checks (auth.go:70), so the player
		cannot interact with the bot anymore.

		Campaign memberships are left untouched, the DM retains sovereignty over their roster and can
		campaign-ban the player separately if desired.

	*/

	if err := models.InsertAuditEntry(r.db, target.ID, invokerID, models.AuditBan, reason); err != nil {
		log.Printf("ban: failed to write audit entry: %v", err)
	}

	log.Printf("ban: %s banned %s (reason: %s)", invokerID, target.ID, reason)
	if reason == "" {
		respond(s, i, fmt.Sprintf(messages.BanSuccessMessage, targetUser.ID))
	} else {
		respond(s, i, fmt.Sprintf(messages.BanSuccessReasonMessage, targetUser.ID, reason))
	}
}

func (r *banCommand) Hidden() bool { return true }
