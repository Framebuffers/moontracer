package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auditlog"
	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		1. A mod/admin invokes `/unban @player`
		2. Auth: invoker must be at least mod.
		3. Load target player, check they are actually banned.
		4. Clear IsBanned + BanReason.
		5. Respond with success or failure.
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

	/*
		Unbanning a member only clears the global ban flag. A Player's membership to a Campaign remains untouched.
	*/
	auditlog.Post(s, u.db, i.GuildID, target.ID, invokerID, models.AuditUnban, "global ban lifted")

	log.Printf("unban: %s unbanned %s", invokerID, target.ID)
	respond(s, i, fmt.Sprintf(messages.UnbanSuccessMessage, targetUser.ID))
}

func (u *unbanCommand) Hidden() bool { return true }
