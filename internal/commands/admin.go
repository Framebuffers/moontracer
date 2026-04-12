package commands

/*
	Flow:
		1. User runs `/admin`.
		2. Authorize: check if the user is a mod or admin.
		3. Show admin hub with action buttons: Manage Campaigns, Active Campaigns (stub), Broadcast (stub), Database, Settings (stub).
*/

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/guard"
	"moontracer/internal/messages"
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
	return &discordgo.InteractionResponseData{
		Content: messages.AdminHubMessage,
		Embeds:  []*discordgo.MessageEmbed{},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.ManageCampaignsCommandDesc,
					Style:    discordgo.PrimaryButton,
					CustomID: messages.BackManageID,
				},
				discordgo.Button{
					Label:    messages.AdminCampaignsLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminCampaignsPrefix,
				},
				discordgo.Button{
					Label:    messages.AdminBroadcastLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminBroadcastPrefix,
				},
			}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.AdminDatabaseLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminDatabasePrefix,
				},
				discordgo.Button{
					Label:    messages.AdminSettingsLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminSettingsPrefix,
				},
				discordgo.Button{
					Label:    messages.AdminDiagLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminDiagPrefix,
				},
			}},
		},
		Flags: discordgo.MessageFlagsEphemeral,
	}
}

// RenderAdminDiag renders the diagnostics sub-view as a message update (from a button click on /admin).
func RenderAdminDiag(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: adminDiagData(s),
	}); err != nil {
		log.Printf("admin_diag: failed to respond: %v", err)
	}
}

/*
adminDiagData uses Components V2 (MessageFlagsIsComponentsV2) because TextDisplay
is a V2-only component.

Under V2, Content and Embeds must NOT be set. Headers go inside TextDisplay blocks.
*/
func adminDiagData(s *discordgo.Session) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: "# " + messages.AdminHubMessage + " — Diagnostics"},
			discordgo.TextDisplay{Content: getGoDiag()},
			discordgo.TextDisplay{Content: getDiscordgoSessionDiag(s)},
			discordgo.TextDisplay{Content: getConfigDiag()},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.BackLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.BackAdminID,
				},
			}},
		},
		Flags: discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
	}
}

func getGoDiag() string {
	version, commit, buildTime := "unknown", "unknown", "unknown"
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			version = bi.Main.Version
		}
		for _, setting := range bi.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) >= 7 {
					commit = setting.Value[:7]
				} else {
					commit = setting.Value
				}
			case "vcs.time":
				buildTime = setting.Value
			}
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	uptime := time.Since(startedAt).Round(time.Second)

	return fmt.Sprintf(`# Runtime Information
**Moontracer**
- Build: %s
- Commit: %s
- Built: %s
- Uptime: %s

## Host
- Hostname: %s
- PID: %d

## Runtime
- Go: %s
- OS/Arch: %s/%s
- CPUs: %d
- Goroutines: %d
- Heap alloc: %d KiB`,
		version, commit, buildTime, uptime,
		hostname, os.Getpid(),
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.NumGoroutine(),
		mem.HeapAlloc/1024)
}

func getDiscordgoSessionDiag(s *discordgo.Session) string {
	userName, userTail := "unknown", "unknown"
	if s.State != nil && s.State.User != nil {
		userName = s.State.User.Username
		uid := s.State.User.ID
		if len(uid) >= 5 {
			userTail = uid[len(uid)-5:]
		} else if uid != "" {
			userTail = uid
		}
	}

	sidTail := "unknown"
	guildCount := 0
	gatewayVersion := 0
	if s.State != nil {
		sid := s.State.SessionID
		if len(sid) >= 5 {
			sidTail = sid[len(sid)-5:]
		}
		guildCount = len(s.State.Guilds)
		gatewayVersion = s.State.Version
	}

	return fmt.Sprintf(`# Session Information
discordgo v%s

## Discordgo Info
- Bot: %s (…%s)
- Gateway latency: **%s**
- Gateway protocol: v%d
- Guilds: %d
- Session ID: …%s
`,
		discordgo.VERSION,
		userName, userTail,
		s.HeartbeatLatency(),
		gatewayVersion,
		guildCount,
		sidTail)
}

func getConfigDiag() string {
	debugAdmin := "(not set)"
	if guard.DebugAdminID != "" {
		debugAdmin = "(set, redacted)"
	}

	adminRole := os.Getenv("ADMIN_ROLE_NAME")
	if adminRole == "" {
		adminRole = "(not set)"
	}
	dbDir := os.Getenv("DB_DIR")
	if dbDir == "" {
		dbDir = "data (default)"
	}
	verbose := os.Getenv("VERBOSE")
	if verbose == "" {
		verbose = "false"
	}

	return fmt.Sprintf(`# Configuration
- Safe mode: **%t**
- Debug admin ID: %s
- Admin role name: %s
- DB dir: %s
- Verbose: %s`,
		guard.SafeMode,
		debugAdmin,
		adminRole,
		dbDir,
		verbose)
}
