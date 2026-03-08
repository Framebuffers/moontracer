package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type addPlayer struct {
	db *bun.DB
}

/*

	One of the issues I've been having, is that every single time someone gets added to a campaign,
	the bot we *currently* use (yagpdb.xyz) pings **every single member** of the campaign.

	And I don't wanna have that happening, it is annoying af.

	This is the plan:
		- A custom type: Notification
		- Notifications have scopes: Member, DM, Mod, Owner.
		- They have an array attached to which members to notify, an empty array means every member of that category.
		- It whitelists by default, starting with the most restrictive access, moving up.
		- Default is DM only. If Players are added, scopes broaden, adding them to each respecitve category.
		- If you want to ping a group, pass the scope and an array, empty if all members are notified, else it notifies the ones inside.

		Checks:
		- Player has roles (registered player, registered mod, admin, mod, etc.)
		- Player is inside Campaign
		- Player has DMs enabled

*/

// Data is the command metadata that Discord shows to users.
func (r *addPlayer) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AddPlayerCommandName,
		Description: messages.AddPlayerCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (r *addPlayer) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
