package interactions

/*
	Schedule conflict flow.

	When a player receives a session announcement DM, a "Schedule conflict" button
	lets them signal to both DMs that they have a clash.

	Flow:
		1. Player clicks "Schedule conflict" on the session DM.
		   CustomID: session_conflict:<guildID>:<sessionID>
		2. Bot queries GetPlayerConflictingSessions (~30min window) and filters out
		   the current campaign. If nothing found, replies with a "no conflicts" message.
		3. A select menu is shown listing the overlapping sessions by campaign + date.
		   CustomID: session_conflict_sel:<guildID>:<thisSessionID>
		4. Player picks the campaign they're going to instead.
		5. Bot:
		   a. Declines the player's RSVP for the current session.
		   b. DMs the absent campaign's DM: "Player won't attend: conflict with Campaign B."
		   c. DMs the present campaign's DM: "Player intends to come: attention re: Campaign A."
		   d. Confirms to the player.

	Both DMs are informed; neither is committed. The DM always has the final say.
*/

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

type sessionConflictHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionConflictHandler) CustomIDPrefix() string { return messages.SessionConflictPrefix }

func (h *sessionConflictHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	guildID := parts[1]
	sessionID := parts[2]
	playerID := helpers.GetUserID(i)

	session, err := db.GetByID[models.Session](h.db, sessionID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	conflicts, err := models.GetPlayerConflictingSessions(h.db, playerID, session.ScheduledAt)
	if err != nil {
		log.Printf("session_conflict: load conflicts for %s: %v", playerID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	// NOTE: do not add the current session itself.
	var others []models.Session
	for _, c := range conflicts {
		if c.ID != sessionID {
			others = append(others, c)
		}
	}

	if len(others) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.SessionConflictNone)
		return
	}

	// build select options
	var options []discordgo.SelectMenuOption
	for _, other := range others {
		campaignName := other.CampaignID
		if other.Campaign != nil {
			campaignName = other.Campaign.Name
		} else if camp, err := db.GetByID[models.Campaign](h.db, other.CampaignID); err == nil {
			campaignName = camp.Name
		}
		label := fmt.Sprintf("%s · %s", campaignName, other.ScheduledAt.UTC().Format("Mon 2 Jan 15:04"))
		if len(label) > 100 {
			label = label[:100]
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: label,
			Value: other.ID,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: messages.SessionConflictPrompt,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    fmt.Sprintf("%s:%s:%s", messages.SessionConflictSelPrefix, guildID, sessionID),
						Placeholder: "Pick the campaign you're going to instead…",
						Options:     options,
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	_ = guildID
}

type sessionConflictSelHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionConflictSelHandler) CustomIDPrefix() string {
	return messages.SessionConflictSelPrefix
}

func (h *sessionConflictSelHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	thisSessionID := parts[2]
	playerID := helpers.GetUserID(i)

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	otherSessionID := values[0]

	thisSession, err := db.GetByID[models.Session](h.db, thisSessionID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	otherSession, err := db.GetByID[models.Session](h.db, otherSessionID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	thisCampaign, err := db.GetByID[models.Campaign](h.db, thisSession.CampaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	otherCampaign, err := db.GetByID[models.Campaign](h.db, otherSession.CampaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	// set that they're not going
	if err := models.UpsertSessionPlayers(h.db, thisSessionID, playerID, models.ResponseDeclined); err != nil {
		log.Printf("session_conflict_sel: decline RSVP for %s/%s: %v", thisSessionID, playerID, err)
	}

	// notify both DMs
	ts := thisSession.ScheduledAt.Unix()
	os := otherSession.ScheduledAt.Unix()

	h.dispatcher.Push(dispatch.DirectMessage{
		ID:      uuid.NewString(),
		Target:  thisCampaign.DungeonMaster,
		Content: fmt.Sprintf(messages.SessionConflictDMToAbsent, playerID, thisCampaign.Name, ts, otherCampaign.Name),
	})

	h.dispatcher.Push(dispatch.DirectMessage{
		ID:      uuid.NewString(),
		Target:  otherCampaign.DungeonMaster,
		Content: fmt.Sprintf(messages.SessionConflictDMToPresent, playerID, otherCampaign.Name, os, thisCampaign.Name),
	})

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.SessionConflictConfirmedFmt, thisCampaign.Name))
}
