package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

// TODO: Implement DM helpers
// SendDM - sends a DM with optional components
// AdminsWithRole - finds guild members with a specific role name

type message struct {
	db *bun.DB
}

func (d *message) CreateChannel(s *discordgo.Session, user_id string) (*discordgo.Channel, error) {
	channel, err := s.UserChannelCreate(user_id)
	if err != nil {
		return channel, nil
	}
	return nil, err
}

func (d *message) SendDM(s *discordgo.Session, ch *discordgo.Channel, content string) (*discordgo.Message, error) {
	msg, err := s.ChannelMessageSend(ch.ID, content)
	if err != nil {
		return msg, err
	}
	return nil, err
}
