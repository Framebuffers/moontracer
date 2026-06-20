# Moontracer v1.1 — Release Notes

## Overview

v1.1 is the first post-launch update. It ships the **Campaign Billboard** system, a full **Campaign Import** workflow, significant **DM quality-of-life** improvements, and patches to all Critical and High-severity findings from the v1.0 security audit.

---

## Security Patches

- **Critical — File handling** (`77a36d0`): Patched a critical vulnerability in file processing that could expose data outside the intended scope.
- **High — `@everyone` role injection** (`4c6c476`): DM-authored content can no longer inject `@everyone`, `@here`, or arbitrary role mentions. Role strings are now allowlisted at point of use.
- **High — Unauthenticated `/importcampaign`** (`e55ad53`): The import command and its confirmation step now require Mod/Admin permissions before any data is written.
- **Medium — Thread permissions** (`322d746`): Fixed thread creation not applying the correct permission overrides for the campaign Discord role.

---

## New Features

### Campaign Billboard

Campaigns are now advertised in a server-wide forum channel, with one post per campaign sorted by format (Campaign / One-Shot / Westmarch).

- Each approved campaign automatically creates or claims a forum post with cover art, schedule, capacity, and a **Join** button (`196e45e`, `b00f05f`).
- Billboard post stays in sync whenever the campaign is edited (`e471a40`).
- **Repost Billboard** action available to admins (`e40806d`) and DMs (see DM Tooling below).
- Admins can configure which forum channel receives which campaign format (`412b54f`).
- Billboard split: campaign info also posted inside the campaign's own channel (`eaf84e8`).
- Category selector: staff choose which Discord category new campaign channels land in (`cd8f961`, `407f58f`).

### Campaign Import (`/importcampaign`)

Allows staff to bring an existing Discord channel under Moontracer management without recreating it.

- Thread-binding UI: select which existing threads map to each bot-recognised slot (`dbf3b03`, `3aa7474`, `79cb7f9`).
- Welcome thread mirrors the billboard and is locked to non-staff on import (`c715cc3`).
- Fixed a DB UNIQUE constraint crash when the campaign had pre-existing threads (`a250e66`).
- Fixed "campaign not found" error when importing a campaign that was already in the DB (`a032764`).

### DM Tooling

- **Thread mapping & remapping**: DMs can bind or rebind channel threads to bot slots at any time via the manage console (`2b70fb8`, `03ee399`).
- **Permission sync**: Sync Discord channel permissions to match the current campaign role without re-creating the channel (`16700f0`).
- **Auto-join thread**: Bot auto-joins newly created threads in the campaign channel (`7af9622`).
- **Set Role** button on the approval DM: DM can assign the campaign Discord role directly from the approval notification (`df28966`).
- **Autoname roles**: When no role name is specified, the bot derives one from the campaign title (`0f23548`).
- **Role cleanup on delete**: Deleting a campaign removes the associated Discord role automatically (`df28966`).

### Session & Schedule

- **`/thisweek` command**: Lists all sessions scheduled for the current week across all campaigns in the guild (`21531c2`).
- **Schedule conflict notification**: Players who join a session that overlaps an existing RSVP are notified at accept time (`6cc26cd`).

### Welcome Thread

The welcome thread now contains:
- Nav links to all active threads (`7af9622`).
- A text-based join instruction alongside the Join button.
- A **Nuke** button (staff only) for emergency channel reset.
- Audit log entry on channel creation.

---

## Improvements & Fixes

| Area | Change |
|---|---|
| Campaign creation | Step 4 added — covers billboard config before approval (`c715cc3`) |
| Campaign delete | Post removed from billboard; channel moved to `archive-<name>` category (`f40bcd7`) |
| Campaign delete | Soft-delete used throughout; historical records kept for staff (`5f4ed9b`) |
| Cooldowns | Buttons and interactions have a per-user cooldown to prevent double-fires (`e40806d`, `e7e182c`) |
| Announcements thread | No cooldown applied in the announcements thread (`21531c2`) |
| Embeds | Fixed formatting on About page, billboard embeds, and schedule field for irregular campaigns (`5623852`, `e7e182c`) |
| Welcome message | Fixed 2000-character limit overflow on large campaign descriptions (`ee688bc`) |
| Modal | Fixed broken modal flow after sequential modal submissions (`c715cc3`) |
| DB | Fixed UNIQUE constraint crash when creating campaigns with duplicate names (`c715cc3`) |
| Join button | Fixed join button disappearing / being overwritten / deleting the billboard post on click (`3b8b7e0`, `631c360`, `feeeb86`) |
| Threads | Bot now pins new threads inside the campaign channel (`ed90aa4`) |
| Search | Fixed campaign search returning no results (`59efcbc`) |
| Cover art | Cover art now renders as a proper Discord embed image (`59efcbc`) |
| Settings | Fixed broken Settings button inside the admin panel (`b00f05f`) |

---

## Known Issues (deferred to v1.2)

- Player session counter on campaign cards is hidden — not yet wired to session data.
- DM-facing unban (campaign-scoped) not yet available.
- Slot count can include the DM as a player — fix shipping in v1.2 (`dev-dmx-slots`).

---

## Upgrade Notes

No manual DB migration required. All schema changes are applied automatically on bot startup.

If you are running a self-hosted instance, pull the latest image and restart. The billboard forum channel must be configured via `/admin Settings` before new campaigns will post to it.
