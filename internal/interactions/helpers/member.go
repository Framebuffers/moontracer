package helpers

import "github.com/bwmarrin/discordgo"

// MemberName returns the guild display name for a user, falling back to userID on API failure.
func MemberName(s *discordgo.Session, guildID, userID string) string {
	m, err := s.GuildMember(guildID, userID)
	if err != nil || m == nil {
		return userID
	}
	return m.DisplayName()
}
