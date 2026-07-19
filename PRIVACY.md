# Privacy Policy

Version 1 - 2026-07-12

---

Moontracer is free and open-source: all of the data processing code, techniques, security updates and disclosure is made public in the [GitHub repo](https://github.com/framebuffers/moontracer). Here are details on how Moontracer stores, processes and transmits your data:

---

## What the bot stores

### Discord user data

- Only your Discord user ID (a number assigned by Discord). We **do not** store your username, display name, avatar, or email.
- Your timezone preference and notification opt-ins. These are **set by you and deletable by you.**
- Your ban status within a server, if applied by a moderator, including the reason.
- Your RSVP status and sessions played count per campaign you've joined.
- Your character sheet URL, if you choose to provide one.

### Campaign and session data

- **Campaign names, descriptions, tags, content warnings, and links.** These are entered by the Dungeon Master.
- **Session schedules and RSVP responses.**
- Discord IDs of **channels, roles, and threads** created for campaigns and post updates.

### Images you upload

- **Player tokens:** the composited output image (.png) is stored under a path that includes your Discord user ID. The original photo and frame are **deleted immediately after processing.**
- **Campaign cover art:** stored similarly, scoped to the campaign.
- Images are served from a private media server under unguessable random filenames. **No public directory listing exists.**

### Audit log
- Staff actions (bans, approvals, etc.) are recorded with who did what, to whom, and when. Entries are **append-only and cannot be edited or deleted.**

---

## What the bot does NOT store

- **Message content:** the bot reads certain messages in memory (e.g. to detect a "join" trigger word on a campaign post) but never persists them.
- **DM content:** the bot sends you DMs for notifications, but does not read or store anything you reply.
- **Voice data:** the bot has no voice capability and requests no voice permissions.
- **Usernames or display names:** only numeric Discord IDs are stored.
- **Email addresses:** none are ever requested or stored.

---

## Discord permissions the bot uses

- **Privileged intents:** this requires approval by Discord for large bots.
- **Server Members Intent:** used to find which members hold a campaign role when importing a campaign, and to sync staff roles on member updates.

### Standard intents
- **Guilds:** read channel, category, and role metadata to create and manage campaign channels.
- **Guild Messages:** detect the join trigger word ("me"/"yo") posted in campaign billboard threads.
- **Direct Messages:** send you session reminders, RSVP confirmations, and moderation notices. The bot does not read your replies.

### Runtime channel/role permissions (these are applied per campaign, not globally)
- Creates and manages Discord roles scoped to each campaign.
- Sets channel permission overwrites to lock campaign channels
- Creates and locks threads (welcome, announcements, sessions, dice rolls) within campaign channels.

---

## Security practices
- **Per-server data isolation:** each Discord server gets its own separate database file. No cross-server data access is possible.
- **Five-tier access control:** every bot action checks your role before executing. A server-wide ban blocks all access unconditionally. It checks roles, from least to most privileged: player, member, DM, mod, admin.
- **Safe Mode:** a deployment flag that disables all Discord mutations (role changes, DMs, channel creation) while keeping the database running, used for maintenance and testing.
- **Media server hardening:** only image file extensions are served, directory listing is disabled, and files are referenced by random UUIDs, not predictable paths.
- **Image validation:** uploaded files are checked for image/* MIME type and capped at 8 MiB before any processing or storage occurs.
- **Append-only audit log:** staff actions are permanently recorded and cannot be retroactively altered.
- **No third-party data sharing:** all data stays on the server running the bot. We do not do analytics, no ad networks, no external APIs beyond Discord itself.