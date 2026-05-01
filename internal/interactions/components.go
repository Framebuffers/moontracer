package interactions

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/manager/models"
)

/*
BuildCampaignSelect builds a StringSelectMenu from a list of campaigns.

customID determines routing (e.g. "campaign_select", "manage_select").
*/
func BuildCampaignSelect(campaigns []models.Campaign, customID, placeholder string) discordgo.SelectMenu {
	var options []discordgo.SelectMenuOption

	for _, c := range campaigns {
		if len(options) >= 25 {
			break // Discord limit
		}

		desc := campaignOptionDescription(c)
		options = append(options, discordgo.SelectMenuOption{
			Label:       truncate(c.Name, 100),
			Value:       c.ID,
			Description: truncate(desc, 100),
		})
	}

	if len(options) == 0 {
		options = append(options, discordgo.SelectMenuOption{
			Label:   "No campaigns available",
			Value:   "none",
			Default: true,
		})
	}

	return discordgo.SelectMenu{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// BuildPlayerCampaignSelect builds a select menu from CampaignPlayer entries.
func BuildPlayerCampaignSelect(entries []models.CampaignPlayer, customID, placeholder string) discordgo.SelectMenu {
	var options []discordgo.SelectMenuOption

	for _, e := range entries {
		if len(options) >= 25 {
			break
		}
		if e.Campaign == nil || !e.Campaign.IsApproved {
			continue
		}
		name := e.Campaign.Name
		desc := fmt.Sprintf("%s — %s", e.Role, e.Status)
		options = append(options, discordgo.SelectMenuOption{
			Label:       truncate(name, 100),
			Value:       e.CampaignID,
			Description: truncate(desc, 100),
		})
	}

	if len(options) == 0 {
		options = append(options, discordgo.SelectMenuOption{
			Label:   "No campaigns",
			Value:   "none",
			Default: true,
		})
	}

	return discordgo.SelectMenu{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

func campaignOptionDescription(c models.Campaign) string {
	format := "Campaign"
	if c.IsOneshot {
		format = "One-shot"
	}
	if c.IsWestmarch {
		format = "Westmarch"
	}

	status := "Open"
	if !c.IsOpen {
		status = "Closed"
	}

	slots := c.DisplaySlots()
	if c.Slots > 0 && c.Slots <= 10 {
		slots = fmt.Sprintf("%s slots", slots)
	}

	return fmt.Sprintf("%s — %s, %s", format, status, slots)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
