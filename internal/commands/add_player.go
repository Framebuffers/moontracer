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

type addPlayer struct {
	db *bun.DB
}

/*
	Flow:
		1. DM invokes `/addplayer @player [tag]`
		2. Look up the Campaign by its tag.
		3. Authorize: invoker must be the DM of that Campaign (sovereignty enforced)
		4. Check if the target is a registered player.
		5. Check if the target isn't already a player on that Campaign.
		6. Checks if there are slots available (Slots == -1 means it has unlimited slots, like a Westmarch, else it counts active players).
		7. Inserts CampaignPlayer with RolePlayer + StatusActive
*/

// Data is the command metadata that Discord shows to users.
func (r *addPlayer) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AddPlayerCommandName,
		Description: messages.AddPlayerCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "player",
				Description: "The player to add to the campaign.",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        messages.TagCommandName,
				Description: messages.TagCommandDesc,
				Required:    true,
			},
		},
	}
}

// Execute is the logic that runs when the user invokes that command.
func (r *addPlayer) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := i.Member.User.ID
	targetUser := i.ApplicationCommandData().Options[0].UserValue(s)
	tag := i.ApplicationCommandData().Options[1].StringValue()

	campaign, err := db.GetByTag[models.Campaign](r.db, tag)
	if err != nil {
		respond(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.CanMutate() {
		respond(s, i, messages.CampaignArchivedMessage)
		return
	}

	ok, err := auth.Authorize(r.db, invokerID, auth.ScopeDM, campaign.ID)
	if err != nil {
		log.Printf("addplayer: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.AddPlayerNotDMOrModMessage)
		return
	}

	registered, err := auth.Authorize(r.db, targetUser.ID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("addplayer: target auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !registered {
		respond(s, i, messages.AddPlayerTargetNotRegistered)
		return
	}

	players, err := models.GetCampaignPlayers(r.db, campaign.ID)
	if err != nil {
		log.Printf("addplayer: error fetching campaign players: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	for _, p := range players {
		if p.PlayerID == targetUser.ID {
			respond(s, i, messages.AddPlayerAlreadyInCampaign)
			return
		}
	}

	if campaign.Slots != -1 {
		activePlayerCount := 0
		for _, p := range players {
			if p.Status == models.StatusActive {
				activePlayerCount++
			}
		}
		if activePlayerCount >= campaign.Slots {
			respond(s, i, messages.AddPlayerCampaignFullMessage)
			return
		}
	}

	cp := &models.CampaignPlayer{
		PlayerID:   targetUser.ID,
		CampaignID: campaign.ID,
		Role:       models.RolePlayer,
		Status:     models.StatusActive,
	}
	if err := db.Insert(r.db, cp); err != nil {
		log.Printf("addplayer: insert failed: %v", err)
		respond(s, i, messages.AddPlayerFailureMessage)
		return
	}

	if campaign.RoleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, targetUser.ID, campaign.RoleID); err != nil {
			log.Printf("addplayer: failed to assign role %s to %s: %v", campaign.RoleID, targetUser.ID, err)
		}
	}

	respond(s, i, fmt.Sprintf(messages.AddPlayerSuccessMessage, targetUser.ID, campaign.Name))
}
