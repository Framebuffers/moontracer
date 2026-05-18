package commands

/*
	Flow:
		1. User runs `/admin`.
		2. Authorize: check if the user is a mod or admin.
		3. Show admin hub with action buttons: Manage Campaigns, Active Campaigns (stub), Broadcast (stub), Database, Settings (stub).
*/

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
startedAt captures an approximation of process start time.

Package-level vars initialize before main(), so this is accurate to within a few ms.
*/
var startedAt = time.Now()

type adminCommand struct {
	db *bun.DB
}

func (a *adminCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AdminCommandName,
		Description: messages.AdminCommandDesc,
	}
}

func (a *adminCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	ok, err := auth.Authorize(a.db, userID, auth.ScopeMod, "")
	if err != nil {
		log.Printf("admin: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.AdminNotStaff)
		return
	}

	RenderAdminHub(s, i)
}

// RenderAdminHub renders the admin panel, callable from the slash command and back buttons.
func RenderAdminHub(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: adminHubData(),
	})
}

// RenderAdminHubUpdate re-renders the admin panel as a message update (for back navigation).
func RenderAdminHubUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: adminHubData(),
	})
}

func adminHubData() *discordgo.InteractionResponseData {
	row1 := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.AdminCampaignsLabel,
			Style:    discordgo.PrimaryButton,
			CustomID: messages.AdminCampaignsPrefix,
		},
		discordgo.Button{
			Label:    messages.AdminBroadcastLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: messages.AdminBroadcastPrefix,
		},
		discordgo.Button{
			Label:    messages.AdminSettingsLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: messages.AdminSettingsPrefix,
		},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: row1},
	}
	if guard.DevMode {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.AdminDiagLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: messages.AdminDiagPrefix,
			},
		}})
	}
	components = append(components, helpers.BackRow(router.ViewMe))

	return &discordgo.InteractionResponseData{
		Content:    messages.AdminHubMessage,
		Embeds:     []*discordgo.MessageEmbed{},
		Components: components,
		Flags:      discordgo.MessageFlagsEphemeral,
	}
}

// BuildTime is set at compile time via -ldflags; remains "unknown" in dev builds.
var BuildTime = "unknown"

// RenderAdminDiag renders the diagnostics sub-view as a message update.
func RenderAdminDiag(s *discordgo.Session, i *discordgo.InteractionCreate, guildDB *bun.DB, guildID string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: adminDiagData(s, guildDB, guildID),
	}); err != nil {
		log.Printf("admin_diag: failed to respond: %v", err)
	}
}

/*
adminDiagData builds a single-code-block diagnostics view.

Using a code fence (not Components V2) keeps the message in V1 so
Back -> admin hub navigation works without a sticky-flags rejection.
*/
func adminDiagData(s *discordgo.Session, guildDB *bun.DB, guildID string) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Content: "```\n" + diagBlock(s, guildDB, guildID) + "```",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				router.BackButton(messages.BackLabel, router.ViewAdmin),
				router.NavButton(messages.HomeLabel, discordgo.SecondaryButton, router.ViewMe),
			}},
		},
		Flags: discordgo.MessageFlagsEphemeral,
	}
}

const diagWidth = 44
const diagLabelW = 12

func drow(label, value string) string {
	return fmt.Sprintf("%-*s %s\n", diagLabelW, label, value)
}

func ddiv() string {
	return strings.Repeat("─", diagWidth) + "\n"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func fmtBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/float64(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%d KiB", n>>10)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func diagBlock(s *discordgo.Session, guildDB *bun.DB, guildID string) string {
	var b strings.Builder

	// header: version + mode flags
	ver := messages.BotVersion
	flags := ""
	if guard.SafeMode {
		flags += " [SAFE]"
	}
	if guard.DevMode {
		flags += " [DEV]"
	}
	header := "moontracer " + ver
	pad := diagWidth - len(header) - len(flags)
	if pad < 1 {
		pad = 1
	}
	b.WriteString(header + strings.Repeat(" ", pad) + flags + "\n")
	b.WriteString(ddiv())

	// runtime
	commit := "unknown"
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range bi.Settings {
			if setting.Key == "vcs.revision" {
				if len(setting.Value) >= 7 {
					commit = setting.Value[:7]
				} else {
					commit = setting.Value
				}
			}
		}
	}
	hostname, _ := os.Hostname()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	b.WriteString(drow("version", ver))
	b.WriteString(drow("commit", commit))
	b.WriteString(drow("built", BuildTime))
	b.WriteString(drow("go", runtime.Version()))
	b.WriteString(drow("os/arch", runtime.GOOS+"/"+runtime.GOARCH))
	b.WriteString(drow("host", hostname))
	b.WriteString(drow("pid", strconv.Itoa(os.Getpid())))
	b.WriteString(drow("uptime", time.Since(startedAt).Round(time.Second).String()))
	b.WriteString(drow("goroutines", strconv.Itoa(runtime.NumGoroutine())))
	b.WriteString(drow("heap", fmt.Sprintf("%d KiB", mem.HeapAlloc/1024)))
	b.WriteString(ddiv())

	// discord session
	userName, userTail, sidTail := "unknown", "?????", "?????"
	guildCount, gatewayVersion := 0, 0
	if s.State != nil {
		if s.State.User != nil {
			userName = s.State.User.Username
			if uid := s.State.User.ID; len(uid) >= 5 {
				userTail = uid[len(uid)-5:]
			}
		}
		if sid := s.State.SessionID; len(sid) >= 5 {
			sidTail = sid[len(sid)-5:]
		}
		guildCount = len(s.State.Guilds)
		gatewayVersion = s.State.Version
	}
	b.WriteString(drow("discordgo", "v"+discordgo.VERSION))
	b.WriteString(drow("gateway", fmt.Sprintf("v%d", gatewayVersion)))
	b.WriteString(drow("bot", fmt.Sprintf("%s (…%s)", userName, userTail)))
	b.WriteString(drow("latency", s.HeartbeatLatency().Truncate(time.Millisecond).String()))
	b.WriteString(drow("guilds", strconv.Itoa(guildCount)))
	b.WriteString(drow("session", "…"+sidTail))
	b.WriteString(ddiv())

	// config
	adminRole := os.Getenv("ADMIN_ROLE_NAME")
	if adminRole == "" {
		adminRole = "(not set)"
	}
	modRole := os.Getenv("MOD_ROLE_NAME")
	if modRole == "" {
		modRole = "(not set)"
	}
	debugAdmin := "(not set)"
	if guard.DebugAdminID != "" {
		debugAdmin = "(set)"
	}
	b.WriteString(drow("safe mode", yesNo(guard.SafeMode)))
	b.WriteString(drow("dev mode", yesNo(guard.DevMode)))
	b.WriteString(drow("verbose", yesNo(os.Getenv("VERBOSE") != "")))
	b.WriteString(drow("admin role", adminRole))
	b.WriteString(drow("mod role", modRole))
	b.WriteString(drow("debug admin", debugAdmin))
	b.WriteString(ddiv())

	// this guild
	if guildDB != nil {
		ctx := context.Background()
		var total, live int
		guildDB.NewSelect().TableExpr("campaigns").ColumnExpr("count(*)").Scan(ctx, &total)
		guildDB.NewSelect().TableExpr("campaigns").Where("is_archived = false").ColumnExpr("count(*)").Scan(ctx, &live)

		campaignStr := strconv.Itoa(live) + " live"
		if archived := total - live; archived > 0 {
			campaignStr += " · " + strconv.Itoa(archived) + " archived"
		}
		b.WriteString(drow("campaigns", campaignStr))

		guildDBSize := int64(0)
		if info, err := os.Stat(filepath.Join("data", guildID+".db")); err == nil {
			guildDBSize = info.Size()
		}
		b.WriteString(drow("db size", fmtBytes(guildDBSize)))
		b.WriteString(ddiv())
	}

	// storage: total across all guilds
	var totalSize int64
	dbFileCount := 0
	if entries, err := os.ReadDir("data"); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".db") {
				if info, err := e.Info(); err == nil {
					totalSize += info.Size()
					dbFileCount++
				}
			}
		}
	}
	totalStr := fmtBytes(totalSize)
	if dbFileCount > 1 {
		totalStr += fmt.Sprintf(" (%d guilds)", dbFileCount)
	}
	b.WriteString(drow("total size", totalStr))

	return b.String()
}

func (a *adminCommand) Hidden() bool { return true }
