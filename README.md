```
*		*	*  * 		*	🌕*
	*	  *		*   *     *
┏┏ ┏━┃┏━┃┏━ ━┏┛┏━┃┏━┃┏━┛┏━┛┏━┃
┃┃┃┃ ┃┃ ┃┃ ┃ ┃ ┏┏┛┏━┃┃  ┏━┛┏┏┛
┛┛┛━━┛━━┛┛ ┛ ┛ ┛ ┛┛ ┛━━┛━━┛┛ ┛
*	   *awoo*	*	*      *
	🐺        *			*    *  
```

# 🐺 Moontracer

A **D&D Campaign Manager Discord bot** for players, DMs and spectators!

Moontracer can manage everything you need to organise your TTRPG experience online: **manage campaigns, players, alerts, and more!**

## 🎲 Quick Start

### 1. Create a Discord application

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications) and create a new application.
2. Under **Bot**, enable these **Privileged Gateway Intents**: `Server Members Intent` and `Message Content Intent`.
3. Copy the **Bot Token**: you'll need it for `DISCORD_BOT_TOKEN`.
4. Under **OAuth2 -> URL Generator**, select scopes `bot` and `applications.commands`, then the permissions your server requires. Use the generated URL to invite the bot.


### 2. Configure

```sh
cp .env.example .env
```

Edit `.env` and fill in at minimum:

```
DISCORD_BOT_TOKEN=your_token_here
ADMIN_ROLE_NAME=Admin        # exact name of the admin role on your server
```

Set `MEDIA_BASE_URL` to the public URL your reverse proxy will serve media from (for example: `https://yourhost.example.com/api/v1/cdn`).

### 3. Run

**With Docker Compose (recommended):**

```sh
docker compose up -d
```

**Locally (requires Go 1.25+):**

```sh
go build ./cmd/moontracer && ./moontracer
```

The bot registers slash commands on startup. With `DISCORD_GUILD_ID` set, commands appear instantly. Without it, global propagation takes up to an hour.

### 4. First use

Once the bot is online in your server, run `/register` to create your player profile, then `/help` to see available commands.

---

## ℹ️ What is it?

- Self-hosted Discord bot to manage TTRPG (mainly Dungeons and Dragons) campaigns in a Discord server.
- Can handle the whole campaign lifecycle: creation, approval, playing and archiving/deleting them.
- Supports Westmarch-style campaigns: multiple concurrent campaign support, players register themselves.
- Staff (admins/mods) approve campaigns, DMs manage their own, players browse and join.

## ⚔️ Main features!

### Campaign Management

- DMs can create new campaigns using `/newcampaign`. 
    - It follows a multi-step modal flow: name your campaign, add a description, system, capacity and schedule.
- Campaigns require staff approval before becoming publicly available.
- When a campaign is approved, a Channel and Threads for general talk, announcements, dice throws and upcoming sessions are created, as well as a (locked and pinned) welcome one, for easier traversal.
- DMs can manage their campaigns using `/manage` with autocomplete, with options for players, sessions, and setting submenus.
- Archive or delete campaigns using `/manage`: it hides/restricts the channel for staff-only.
- Upload cover art using `/uploadcover`: self-hosted, independent from Discord's CDN.
- Browse campaigns using `/campaigns` (menu-based), or `/search` (text autocomplete).

### Session scheduling:

- DMs schedule their sessions via `/newsession` (modal: date DD/MM/YYYY, time, and optional notes).
- Sessions announce to the campaign's `#announcements` thread with a role mention.
- Session embed includes: title, date/time, DM, capacity, Response counts (who's going/not going/on a waitlist).
- Players can say if they're going or not via a button on their DMs or on the `#announcements` message.
- Conflict detection: warns when a player already has a session scheduled at the same time.
- Session reminders: if the player opts-in, they can receive notifications 1 hour before a session starts.
- `/nextsessions` show a player's upcoming sessions across all their campaigns, sorted by date, in their local timezone.

### Player management

- Players register using `/register`
- `/me` hub: a centre for your campaigns, sessions, tokens and settings.
- Players can join campaigns from either the search box command, or via the UI.
- Capacity enforcement: sessions reject new players when they're full.
- Per-player timezone settings (IANA tz names): all times and dates are formatted in their local timezones.

### Tokens

- Upload and create new player tokens using `/newtoken`.
    - Upload your image, add a frame **or** write a colour of your choosing.
    - The bot composites a new circular token based on the image you give it.
    - Returns a PNG ready to use!
- Flow: preview your token, set a name, save it, assign it to a campaign or skip.
- See your tokens using `/tokens`: a gallery of all your uploaded tokens. Preview them as embeds, select them with a menu.
- Download tokens: get your tokens using a one-time download URL (UUID claim, 10min TTL). It never exposes internal paths for extra security!
- You can reassign tokens to different campaigns after creation.

### Staff/Admin

- `/admin` panel (hidden from `/help`, staff-only): They hold server-wide settings, with per-guild role name overrides.
- `/ban` and `/unban` (hidden from `/help`, staff-only): ban players server-wide.
- `/campaigndatabase`: see all the campaigns to have ever been created inside your server.
- `DEBUG_ADMIN_ID` env var enables a two-factor developer override for sovereignty operations: if and only if the Discord account has permissions inside a server, it can perform operations on it. Even if it remains set up, nobody can enter your server without your consent, by design.
- `SAFE_MODE` env var disables all Discord-mutating operations (like assigning roles, creating channels, etc): perfect for testing!

## ⌨️ Command List
| Command | Who | Description |
|---|---|---|
| `/register` | Anyone | Register as a player |
| `/me` | Player | Profile hub |
| `/campaigns` | Anyone | Browse all campaigns |
| `/search` | Anyone | Search campaigns by name |
| `/campaign` | Anyone | Show campaign details |
| `/newcampaign` | Player | Create a campaign (you become DM) |
| `/manage` | DM | Manage one of your campaigns (autocomplete) |
| `/newsession` | DM | Schedule and announce a session |
| `/nextsessions` | Player | View upcoming sessions |
| `/tokens` | Player | Browse and manage your tokens |
| `/newtoken` | Player | Create a player token |
| `/uploadcover` | DM | Upload campaign cover art |
| `/abandon` | DM | Archive your campaign permanently |
| `/help` | Anyone | List all commands |
| `/about` | Anyone | About the bot |
| `/admin` | Staff | Mod/admin panel (hidden from help) |
| `/ban` / `/unban` | Staff | Global player ban (hidden from help) |

## 🏰 Architecture

- Written in **Go** (module: `github.com/framebuffers/moontracer`)
- Discord library: [discordgo](https://github.com/bwmarrin/discordgo) (bwmarrin).
- ORM: [uptrace/bun](https://github.com/uptrace/bun) with SQLite per-guild (modernc.org/sqlite).
- One SQLite database file per guild, stored in a bind-mounted /app/data directory.
- Embedded IANA timezone database (time/tzdata): no system tzdata dependency.
- Image processing: [disintegration/imaging](https://github.com/disintegration/imaging), [fogleman/gg](https://github.com/fogleman/gg).
- Built-in HTTP media server (Go stdlib net/http) for serving cover art and tokens.
- One-time opaque download endpoint (claim store, UUID, 10-min TTL).
- All Discord-mutating operations wrapped in a guard package that respects SAFE_MODE.
- Command registration uses ApplicationCommandBulkOverwrite- stale commands auto-removed on startup.
- Global/guild command mode: 
    - guild ID set -> instant registration (dev); 
    - unset -> global (up to 1h propagation).

## ⚙️ Configuration/Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DISCORD_BOT_TOKEN` | Yes |- | Bot token from Discord Developer Portal |
| `ADMIN_ROLE_NAME` | Yes |- | Name of the admin/owner role on the server |
| `MOD_ROLE_NAME` | No |- | Name of the moderator role (optional) |
| `DISCORD_GUILD_ID` | No |- | Limit command registration to one guild (dev mode) |
| `MEDIA_PORT` | No | `8090` | Port for the built-in media server |
| `MEDIA_BASE_URL` | No | `http://localhost:8090/api/v1/cdn` | Public CDN root URL |
| `MEDIA_DOWNLOAD_URL` | No | `http://localhost:8090` | Public server root for `/dl/` download links |
| `SAFE_MODE` | No |- | Disable Discord mutations (roles, DMs) |
| `DEV_MODE` | No |- | Development mode flag |
| `DEBUG_ADMIN_ID` | No |- | Developer override Discord user ID |
| `VERBOSE` | No |- | Enable verbose logging with timestamps |

## 📦 Deployment

- Docker: Dockerfile + docker-compose.yml provided.
- Designed to sit behind a reverse proxy (Caddy / Traefik / nginx) for TLS and rate limiting.
- Data volume: bind-mount a host directory to /app/data (contains all per-guild SQLite DBs and media files).
- No external services required: fully self-contained (SQLite, no Redis, no external DB).
- Docker Compose network: joins a named web_proxy network (configurable via `MOONTRACER_PROXY_NETWORK`)

## 🛡️ Security Notes

- Download links use opaque UUID tokens (10-minute TTL, one-time use): internal paths never exposed.
- Path traversal rejected on /dl/ endpoint.
- Staff-only commands hidden from /help listing.
- `SAFE_MODE` prevents accidental Discord mutations in staging environments.
- `ADMIN_ROLE_NAME` required at startup: bot refuses to start without it.
- Recommend fronting the media server with a reverse proxy (rate limiting, TLS).

## 🏗️ Development Notes

- Run with `DISCORD_GUILD_ID` set for instant command registration during development.
- `SAFE_MODE=true` lets you run against a real DB without sending Discord roles/DMs.
- `VERBOSE=true` enables microsecond-precision structured logs with file/line info.
- Tests: `go test ./...` (ban, token upload, admin visibility covered).
- Build: `go build ./...`: .silent output means success.

## 🔐 Operation Security: how can you make sure this bot is not backdoored

`DEBUG_ADMIN_ID` is an operator-controlled env var for development. 

In production (SAFE_MODE off), it requires the invoking user to also hold the configured admin Discord role: the env var alone grants **nothing**. 
It is never hardcoded. Operators who don't need it can leave it unset.

## 📄 License

[GNU Affero General Public License v3.0](LICENSE)

You're **free** to self-host, modify, and redistribute; but any publicly-accessible network service built on this code **must also be open source under the same license.**

## 🤝 Contributing

Issues and pull requests are welcome! Please open an issue first for significant changes, so we can discuss the approach before you invest time in it.

## 💗 Thanks!

Thanks to the folks at the r/Chile D&D Discord server for letting me test live on prod on their server, and giving me the idea and support to keep on going. Here onto more nat-d20's!

🐺 awoo! 🌕