package discord

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/auth"
	"moontracer/internal/commands"
	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/guard"
	"moontracer/internal/scheduler"
)

/*
Bot wraps a discordgo session and manages its lifecycle.

Note: guildID is optional: if it is set, commands register to that guild only (dev mode)
*/
type Bot struct {
	session      *discordgo.Session
	guildID      string
	guildDBM     *db.GuildDBManager
	adminRole    string
	modRole      string
	registered   []*discordgo.ApplicationCommand
	dispatcher   *dispatch.Dispatcher
	scheduler    *scheduler.Scheduler
	dataDir      string
	mediaBaseURL string
}

// New creates a Bot with the given token, guild ID, role names, and guild DB manager.
func New(token, guildID, adminRole, modRole, dataDir, mediaBaseURL string, guildDBM *db.GuildDBManager) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	return &Bot{
		session:      s,
		guildID:      guildID,
		guildDBM:     guildDBM,
		adminRole:    adminRole,
		modRole:      modRole,
		dataDir:      dataDir,
		mediaBaseURL: mediaBaseURL,
	}, nil
}

/*
	Flow (Run):
		1. Add the main event handler to the session (per-guild DB resolution + command dispatch).
		2. Open the Discord gateway connection.
		3. Discover all guilds the bot is in, initialize per-guild databases in parallel.
		4. Sync server roles for each guild in parallel.
		5. Register all commands globally with Discord.
		6. Block waiting for SIGINT/SIGTERM (Ctrl+C).
		7. On shutdown signal, in dev mode (guildID set) remove the guild-scoped
		   commands from Discord; in production (global registration) leave them
		   in place so restarts don't wipe commands from every guild.
		8. Close the gateway connection, close all guild databases, and return.
*/

/*
Run opens the gateway, registers global commands, blocks until
SIGINT/SIGTERM, then removes commands and closes the session.
*/
func (b *Bot) Run() error {
	// workers
	b.dispatcher = dispatch.NewDispatcher(b.session, 5)
	b.scheduler = scheduler.New(b.guildDBM, b.dispatcher)

	b.session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages

	b.session.AddHandler(NewHandler(b.guildDBM, b.dispatcher, b.adminRole, b.scheduler, b.dataDir, b.mediaBaseURL))

	b.session.AddHandler(func(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
		guildDB, err := b.guildDBM.GetOrCreate(e.GuildID)
		if err != nil {
			log.Printf("bot: warning: could not get DB for guild %s on member update: %v", e.GuildID, err)
			return
		}
		if err := auth.SyncServerRoles(guildDB, s, e.GuildID, b.adminRole, b.modRole); err != nil {
			log.Printf("bot: warning: role sync on member update failed for guild %s: %v", e.GuildID, err)
		}
	})

	b.session.AddHandler(HandleGuildMemberRemove(b.guildDBM, b.scheduler))

	// Initialize new guilds joined mid-runtime.
	b.session.AddHandler(func(s *discordgo.Session, e *discordgo.GuildCreate) {
		if guard.DebugGuildID != "" && e.Guild.ID != guard.DebugGuildID {
			log.Printf("bot: skipping guild %s (%s) — scoped to %s", e.Guild.Name, e.Guild.ID, guard.DebugGuildID)
			return
		}
		guildDB, err := b.guildDBM.GetOrCreate(e.Guild.ID)
		if err != nil {
			log.Printf("bot: failed to create DB for guild %s (%s): %v", e.Guild.Name, e.Guild.ID, err)
			return
		}
		if err := commands.RegisterCommands(guildDB, nil); err != nil {
			log.Printf("bot: failed to register commands in DB for guild %s: %v", e.Guild.ID, err)
		}
		if err := auth.SyncServerRoles(guildDB, s, e.Guild.ID, b.adminRole, b.modRole); err != nil {
			log.Printf("bot: failed to sync roles for guild %s: %v", e.Guild.ID, err)
		}
		log.Printf("bot: initialized guild %s (%s)", e.Guild.Name, e.Guild.ID)
	})

	if err := b.session.Open(); err != nil {
		return err
	}
	defer b.session.Close()

	b.dispatcher.Start()

	appID := b.session.State.User.ID
	log.Printf("bot: logged in as %s (app %s)", b.session.State.User.Username, appID)

	/*
		Discover all guilds and initialize their databases in parallel.

		NOTE:
			In dev mode scoped to a single guild, ignore all other guilds.
	*/
	var guildIDs []string
	for _, g := range b.session.State.Guilds {
		if guard.DebugGuildID != "" && g.ID != guard.DebugGuildID {
			log.Printf("bot: skipping guild %s (%s) — scoped to %s", g.Name, g.ID, guard.DebugGuildID)
			continue
		}
		guildIDs = append(guildIDs, g.ID)
	}
	log.Printf("bot: discovered %d guild(s), initializing databases...", len(guildIDs))
	b.guildDBM.InitForGuilds(guildIDs)
	b.scheduler.BootScan(guildIDs)

	// Register command metadata in each guild's DB and sync roles in parallel.
	var wg sync.WaitGroup
	for _, g := range b.session.State.Guilds {
		if guard.DebugGuildID != "" && g.ID != guard.DebugGuildID {
			continue
		}
		wg.Add(1)
		go func(guild *discordgo.Guild) {
			defer wg.Done()
			guildDB, err := b.guildDBM.GetOrCreate(guild.ID)
			if err != nil {
				log.Printf("bot: skipping guild %s: %v", guild.ID, err)
				return
			}
			if err := commands.RegisterCommands(guildDB, nil); err != nil {
				log.Printf("bot: failed to register commands in DB for guild %s: %v", guild.ID, err)
			}
			if err := auth.SyncServerRoles(guildDB, b.session, guild.ID, b.adminRole, b.modRole); err != nil {
				log.Printf("bot: role sync failed for guild %s: %v", guild.ID, err)
			}
		}(g)
	}
	wg.Wait()
	log.Println("bot: all guild databases initialized and roles synced")

	if err := b.registerCommands(appID); err != nil {
		return err
	}

	log.Println("bot is running — press Ctrl+C to exit")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	b.scheduler.Stop()
	b.dispatcher.Stop()
	if b.guildID != "" {
		b.removeCommands(appID)
	} else {
		log.Println("bot: leaving global commands registered across restart")
	}
	b.guildDBM.Close()

	return nil
}

func (b *Bot) registerCommands(appID string) error {
	/*
		Flow:
			Use the first available guild's DB to build command metadata.
			commands.All needs a *bun.DB for struct init, but registerCommands only calls cmd.Data() for the Discord API registration.
	*/
	var guildIDs []string
	for _, g := range b.session.State.Guilds {
		guildIDs = append(guildIDs, g.ID)
	}
	if len(guildIDs) == 0 {
		log.Println("bot: no guilds found, skipping command registration")
		return nil
	}

	bunDB, err := b.guildDBM.GetOrCreate(guildIDs[0])
	if err != nil {
		return err
	}

	/*
		When scoped to a single guild (dev mode), clear any stale global registrations
		left over from earlier runs so Discord doesn't serve them in other servers.
	*/
	if b.guildID != "" {
		existing, err := b.session.ApplicationCommands(appID, "")
		if err != nil {
			log.Printf("bot: warning: could not list global commands for cleanup: %v", err)
		} else {
			for _, g := range existing {
				if err := b.session.ApplicationCommandDelete(appID, "", g.ID); err != nil {
					log.Printf("bot: warning: failed to delete stale global /%s: %v", g.Name, err)
				} else {
					log.Printf("bot: deleted stale global /%s", g.Name)
				}
			}
		}
	}

	/*
		BulkOverwrite atomically replaces all registered commands with the current list.
		Any command previously registered but absent from All() is automatically removed.

		No manual cleanup needed when a command is retired.
	*/
	all := commands.All(bunDB, b.dispatcher, b.dataDir, b.mediaBaseURL)
	cmdData := make([]*discordgo.ApplicationCommand, len(all))
	for i, cmd := range all {
		cmdData[i] = cmd.Data()
	}

	registered, err := b.session.ApplicationCommandBulkOverwrite(appID, b.guildID, cmdData)
	if err != nil {
		return err
	}
	b.registered = registered

	scope := "global"
	if b.guildID != "" {
		scope = "guild " + b.guildID
	}
	for _, cmd := range registered {
		log.Printf("bot: registered /%s (%s)", cmd.Name, scope)
	}
	return nil
}

func (b *Bot) removeCommands(appID string) {
	for _, cmd := range b.registered {
		if err := b.session.ApplicationCommandDelete(appID, b.guildID, cmd.ID); err != nil {
			log.Printf("bot: failed to remove /%s: %v", cmd.Name, err)
		} else {
			log.Printf("bot: removed /%s", cmd.Name)
		}
	}
}
