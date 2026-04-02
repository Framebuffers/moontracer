package commands

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. DM invokes `/abandon tag:X`
		2. Auth: only the DM of that campaign can archive it (sovereignty enforced).
		3. Set IsArchived = true, ArchivedAt = now, ArchivedReason = "DM abandoned".
		4. Set all CampaignPlayer statuses to StatusFinished.
		5. Record an audit entry.
*/

type abandonCampaign struct {
	db *bun.DB
}

func (a *abandonCampaign) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AbandonCommandName,
		Description: messages.AbandonCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        messages.TagCommandName,
				Description: messages.TagCommandDesc,
				Required:    true,
			},
		},
	}
}

func (a *abandonCampaign) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := i.Member.User.ID
	tag := i.ApplicationCommandData().Options[0].StringValue()

	campaign, err := db.GetByTag[models.Campaign](a.db, tag)
	if err != nil {
		respond(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.CanMutate() {
		respond(s, i, messages.CampaignArchivedMessage)
		return
	}

	// Auth: only the DM can abandon their own campaign.
	ok, err := auth.Authorize(a.db, invokerID, auth.ScopeDM, campaign.ID)
	if err != nil {
		log.Printf("abandon: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.AbandonNotDMMessage)
		return
	}

	if err := ArchiveCampaign(a.db, campaign, messages.AbandonReasonDM); err != nil {
		log.Printf("abandon: failed to archive campaign %s: %v", campaign.ID, err)
		respond(s, i, messages.AbandonFailureMessage)
		return
	}

	if err := models.InsertAuditEntry(a.db, invokerID, invokerID, models.AuditCampaignArchive, fmt.Sprintf("abandoned campaign %s (%s)", campaign.Name, campaign.Tag)); err != nil {
		log.Printf("abandon: failed to write audit entry: %v", err)
	}

	log.Printf("abandon: %s archived campaign %s (%s)", invokerID, campaign.Name, campaign.ID)
	respond(s, i, fmt.Sprintf(messages.AbandonSuccessMessage, campaign.Name))
}

// ArchiveCampaign sets a campaign as archived and marks all members as finished.
// Shared between the /abandon command and the GuildMemberRemove event handler.
func ArchiveCampaign(database *bun.DB, campaign *models.Campaign, reason string) error {
	campaign.IsArchived = true
	campaign.ArchivedAt = time.Now().UTC()
	campaign.ArchivedReason = reason

	if err := db.Update(database, campaign); err != nil {
		return fmt.Errorf("update campaign: %w", err)
	}

	// Set all campaign members to finished.
	players, err := models.GetCampaignPlayers(database, campaign.ID)
	if err != nil {
		return fmt.Errorf("get campaign players: %w", err)
	}

	for _, p := range players {
		if p.Status == models.StatusActive || p.Status == models.StatusPending || p.Status == models.StatusHiatus {
			if err := models.SetCampaignPlayerStatus(database, p.PlayerID, campaign.ID, models.StatusFinished); err != nil {
				log.Printf("archive: failed to set player %s to finished in campaign %s: %v", p.PlayerID, campaign.ID, err)
			}
		}
	}

	return nil
}
