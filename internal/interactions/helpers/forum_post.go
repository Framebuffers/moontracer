package helpers

import (
	"fmt"
	"math"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
	"github.com/uptrace/bun"
)

/*
NewCampaignForumPost formats a Campaign into a message to be sent to a forum.

This is used to create a "campaign billboard" of new games being created by the players.
*/
func NewCampaignForumPost(db *bun.DB, s *discordgo.Session, c *models.Campaign) (title, body, coverURL string) {
	if !c.IsApproved || c.IsArchived {
		return "", "", ""
	}

	format := messages.ForumPostFormatCampaign
	if c.IsOneshot {
		format = messages.ForumPostFormatOneshot
	} else if c.IsWestmarch {
		format = messages.ForumPostFormatWestmarch
	}

	schedule := messages.ForumPostScheduleUnset
	if c.Schedule.HasSchedule() {
		schedule = fmt.Sprintf("%ss at %s UTC (%s, %.0fh sessions)",
			c.Schedule.DayName(),
			c.Schedule.StartTime,
			c.Schedule.Frequency,
			c.Schedule.DurationHours,
		)
	}

	status := messages.ForumPostStatusClosed
	if c.IsOpen {
		status = messages.ForumPostStatusOpen
	}

	active := activeCampaignPlayers(c.CampaignPlayers)
	confirmed, waiting := splitPlayers(active, c.Slots)

	var slotsLine string
	if c.Slots <= 0 || c.Slots == math.MaxInt32 {
		slotsLine = messages.ForumPostSlotsUnlimited
	} else {
		slotsLine = fmt.Sprintf("%d/%d", len(active), c.Slots)
	}

	books := strings.Join(c.Game.BooksAllowed, ", ")
	if books == "" {
		books = messages.NoneLabel
	}
	extras := strings.Join(c.Game.OtherGame, ", ")
	if extras == "" {
		extras = messages.NoneLabel
	}

	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", c.Name)
	fmt.Fprintf(&b, "%s\n\n", c.Description)

	fmt.Fprintf(&b, "## Campaign details\n\n")
	fmt.Fprintf(&b, "**DM:** <@%s>\n", c.DungeonMaster)
	fmt.Fprintf(&b, "**Format:** %s\n", format)
	fmt.Fprintf(&b, "**Schedule:** %s\n\n", schedule)

	fmt.Fprintf(&b, "## Format\n\n")
	fmt.Fprintf(&b, "- Edition: %s\n", c.Game.Edition)
	fmt.Fprintf(&b, "- Rules: %s\n", c.Game.Rules)
	fmt.Fprintf(&b, "- VTT: %s\n", c.Game.VTT)
	fmt.Fprintf(&b, "- Books: %s\n", books)
	fmt.Fprintf(&b, "- Extras: %s\n\n", extras)

	if len(c.Warnings) > 0 {
		fmt.Fprintf(&b, "## Trigger warnings\n\n%s\n\n", strings.Join(c.Warnings, ", "))
	}

	if c.Extra != "" {
		fmt.Fprintf(&b, "## Extra info\n\n%s\n\n", c.Extra)
	}

	fmt.Fprintf(&b, "---\n\n")
	fmt.Fprintf(&b, "## Players\n\n")
	fmt.Fprintf(&b, "- **Status:** %s\n", status)
	fmt.Fprintf(&b, "- **Slots:** %s\n", slotsLine)

	if len(confirmed) > 0 {
		fmt.Fprintf(&b, "- **Confirmed players:** %s\n", strings.Join(playerMentions(confirmed), ", "))
	} else {
		fmt.Fprintf(&b, "- **Confirmed players:** %s\n", messages.ForumPostNoPlayers)
	}

	if len(waiting) > 0 {
		fmt.Fprintf(&b, "- **Waiting list:** %s\n", strings.Join(playerMentions(waiting), ", "))
	}

	if cover, err := models.MediaByCampaign(db, c.ID, models.KindCoverArt); err == nil && len(cover) > 0 {
		coverURL = cover[0].URL
	}

	return c.Name, b.String(), coverURL
}

/*
UpdateBillboard edits the starter message of a campaign's billboard forum thread
with freshly rendered content. No-op if the campaign has no thread yet.

The starter message of a Discord forum thread has the same ID as the thread channel itself,
so ChannelMessageEdit(threadID, threadID, body) is the correct call.
*/
func UpdateBillboard(s *discordgo.Session, db *bun.DB, c *models.Campaign) error {
	if c.BillboardThreadID == "" {
		return nil
	}
	_, body, coverURL := NewCampaignForumPost(db, s, c)
	if body == "" {
		return nil
	}
	edit := discordgo.NewMessageEdit(c.BillboardThreadID, c.BillboardThreadID).SetContent(body)
	if coverURL != "" {
		edit.SetEmbed(&discordgo.MessageEmbed{Image: &discordgo.MessageEmbedImage{URL: coverURL}})
	} else {
		embeds := []*discordgo.MessageEmbed{}
		edit.Embeds = &embeds
	}
	components := BillboardComponents(c)
	edit.Components = &components
	_, err := s.ChannelMessageEditComplex(edit)
	return err
}

/*
BillboardComponents returns the action row for the billboard starter message.

Shows a Join button when the campaign is open. Returns an empty slice otherwise.
*/
func BillboardComponents(c *models.Campaign) []discordgo.MessageComponent {
	if !c.IsOpen {
		return []discordgo.MessageComponent{}
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.JoinCampaignLabel,
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("campaign_join:%s", c.Tag),
			},
		}},
	}
}

func activeCampaignPlayers(players []models.CampaignPlayer) []models.CampaignPlayer {
	var out []models.CampaignPlayer
	for _, cp := range players {
		if cp.BannedFromCampaign || cp.Status == models.StatusBanned {
			continue
		}
		if cp.RSVPStatus == models.RSVPDeclined {
			continue
		}
		if cp.Status != models.StatusActive {
			continue
		}
		out = append(out, cp)
	}
	return out
}

func splitPlayers(players []models.CampaignPlayer, slots int) (confirmed, waiting []models.CampaignPlayer) {
	if slots <= 0 || slots == math.MaxInt32 || len(players) <= slots {
		return players, nil
	}
	return players[:slots], players[slots:]
}

func playerMentions(players []models.CampaignPlayer) []string {
	out := make([]string, len(players))
	for i, cp := range players {
		out[i] = fmt.Sprintf("<@%s>", cp.PlayerID)
	}
	return out
}
