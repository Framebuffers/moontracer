package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"moontracer/internal/mediaserver"
	"moontracer/internal/messages"
)

/*
uploadTokenCommand implements /uploadtoken.

	Flow:
		1. User runs /uploadtoken source:<photo> frame:<frame>.
		2. Bot defers the response.
		3. Downloads both attachments to temp paths.
		4. Calls mediaserver.ProcessToken.
		5. Edits the deferred response with a preview embed + Apply/Discard buttons.
		6. Apply: saves the Media record and removes temps.
				Discard: removes all three temp files.
*/
type uploadTokenCommand struct {
	db           *bun.DB
	dataDir      string
	mediaBaseURL string
}

func (c *uploadTokenCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.TokenUploadCommandName,
		Description: messages.TokenUploadCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        messages.TokenUploadSourceOptName,
				Description: messages.TokenUploadSourceOptDesc,
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        messages.TokenUploadFrameOptName,
				Description: messages.TokenUploadFrameOptDesc,
				Required:    true,
			},
		},
	}
}

func (c *uploadTokenCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	data := i.ApplicationCommandData()
	var sourceID, frameID string
	for _, opt := range data.Options {
		switch opt.Name {
		case messages.TokenUploadSourceOptName:
			sourceID = opt.Value.(string)
		case messages.TokenUploadFrameOptName:
			frameID = opt.Value.(string)
		}
	}

	source := data.Resolved.Attachments[sourceID]
	frame := data.Resolved.Attachments[frameID]
	if source == nil || frame == nil {
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !strings.HasPrefix(source.ContentType, "image/") || !strings.HasPrefix(frame.ContentType, "image/") {
		respond(s, i, messages.TokenUploadNotImage)
		return
	}
	if source.Size > maxCoverBytes || frame.Size > maxCoverBytes {
		respond(s, i, messages.TokenUploadTooLarge)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})

	token := uuid.NewString()
	sourceExt := extOrDefault(source.Filename, ".jpg")
	frameExt := extOrDefault(frame.Filename, ".png")

	sourceDsk, _ := mediaserver.TokenPath(c.dataDir, c.mediaBaseURL, i.GuildID, userID, "src_"+token, sourceExt)
	frameDsk, _ := mediaserver.TokenPath(c.dataDir, c.mediaBaseURL, i.GuildID, userID, "frm_"+token, frameExt)
	outDisk, outURL := mediaserver.TokenPath(c.dataDir, c.mediaBaseURL, i.GuildID, userID, "out_"+token, ".png")

	if _, err := mediaserver.Download(source.URL, sourceDsk); err != nil {
		log.Printf("uploadtoken: download source failed for %s: %v", userID, err)
		editDeferred(s, i, messages.TokenUploadProcessFailed)
		return
	}
	if _, err := mediaserver.Download(frame.URL, frameDsk); err != nil {
		log.Printf("uploadtoken: download frame failed for %s: %v", userID, err)
		cleanupFiles(sourceDsk)
		editDeferred(s, i, messages.TokenUploadProcessFailed)
		return
	}

	if err := mediaserver.ProcessToken(sourceDsk, frameDsk, outDisk); err != nil {
		log.Printf("uploadtoken: processing failed for %s: %v", userID, err)
		cleanupFiles(sourceDsk, frameDsk)
		editDeferred(s, i, messages.TokenUploadProcessFailed)
		return
	}

	// format: token_apply:{guildID}:{userID}:{token}
	applyID := fmt.Sprintf("%s:%s:%s:%s", messages.TokenApplyPrefix, i.GuildID, userID, token)
	discardID := fmt.Sprintf("%s:%s:%s:%s", messages.TokenDiscardPrefix, i.GuildID, userID, token)

	embed := &discordgo.MessageEmbed{
		Title: "Token Preview",
		Color: messages.EmbedColor,
		Image: &discordgo.MessageEmbedImage{URL: outURL},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: strPtr(messages.TokenUploadPreviewContent),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: messages.TokenApplyLabel, Style: discordgo.SuccessButton, CustomID: applyID},
				discordgo.Button{Label: messages.TokenDiscardLabel, Style: discordgo.DangerButton, CustomID: discardID},
			}},
		},
	})
}

func extOrDefault(filename, def string) string {
	if ext := filepath.Ext(filename); ext != "" {
		return ext
	}
	return def
}

func cleanupFiles(paths ...string) {
	for _, p := range paths {
		os.Remove(p)
	}
}

func editDeferred(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: strPtr(content)})
}

func strPtr(s string) *string { return &s }
