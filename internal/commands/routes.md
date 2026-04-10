# Moontracer Menus

Note: This route structure follows the same rule: Each interaction has buttons for the one down the list. e. g: using `/register` opens an interaction where it has buttons for Join, and Browse. Clicking on Brose opens an interaction with buttons for "By Format" (clicking it displays buttons for all three options) or "All".

Interactions can also have commands associated with them, some being links to pre-existing interactions. This is done such that the user has a logical flow of where things are, without having to backtrack.

Each interaction also has a back button to go up the directory.

Reference for syntax:

- Dropdowns: Select Menus
- [thing]:[option] text argument


## Onboarding

- Register [/register]
    - Join a Campaign [/join]
    - Browse Campaigns [/campaigns]
        - By Format:
            - One-shots [/campaigns format:oneshot]
            - Westmarches [/campaigns format:westmarch]
            - Campaigns [/campaigns format:campaign]
        - All

## Player

### Player Menu [/me]

- My Campaigns [/mycampaigns]
- Next Sessions [/nextsessions]
- Notifications [/notifications] // if something is pending, got approved, denied, a mod sent a message

## CampaignPlayer [/campaign dropdown:campaign]

- Tokens
    - Add or Create a new token [/token dropdown:campaign]
- Next session [/nextsession dropdown:campaign]
- Send a message to the DM [/messagedm dropdown:campaign]
- Leave Campaign

## Campaign [/campaign dropdown:campaign]

- Ask to join [button]
- Next Session [button]
- Stats [button, /stats dropdown:campaign]
    - Deaths
    - Sessions Played
    - Journal [/gamejournal dropdown:campaign]


## DM [/mycampaigns (if is DM)]

- Approve new players [/approve dropdown:campaign]
- New Campaign [/newcampaign, opens modal]
- Requests [/requests, opens requests to join session]
- Edit my Campaigns [/configcampaign dropdown:campaign]
- Config [/configcampaign dropdown:campaign]
    - Edit [/editcampaign dropdown:campaign]
        - Journal [/editjournal dropdown:campaign]
            - Add entry [/newjournal dropdown:campaign modal:text]
            - Edit entry [/editjournal dropdown:campaign entry_number]
            - Entries [/journal dropdown:campaign]
            - Config [/configjournal dropdown:campaign]
                - Disable [/disablejournal dropdown:campaign]
                - Hide from players [/hidejournal dropdown:campaign]
        - Synopsis [/campaignsynopsis dropdown:campaign]
        - Cover art [/editcoverart dropdown:campaign]
        - Links [/campaignlinks dropdown:campaign]
            - Add [/addlink dropdown:campaign link]
            - [a list of all links]
                - Edit [/editlink dropdown:campaign select:link_key value]
                - Remove [/removelink dropdown:campaign select:link_key]
                - [show value on interaction] // usuals are: [vtt, music, character_sheets (a link to dndbeyond, or wherever your character sheet exists)]
    - Players [/myplayers dropdown:campaign]
        - List
        - Remove player from next session [/kickfromsession campaign:currently_selected player:player]
        - Tokens [/campaigntokens dropdown:campaign]
            - Download all tokens [/downloadtokens dropdown:campaign]
            - Change Player's token [/changetoken campaign:currently_selected player:player file]
            - Prioritize new players [/prioritizenewplayers dropdown:campaign yes/no] 
                /* on westmarches, the DM assigns a set number of players per session (because they cannot manage infinite number of players). 
                
                to manage who gets to play, a DM can prioritise by: queue (fill the quota and then close), join date/least plays (effectively, the ones that have played the least get priority over the ones that have played the most, in order to avoid having the same party members each time).
                */
    - Scheduling [/campainschedule dropdown:campaign]
        - Set frequency [/setschedule campaign:currently_active frequency:{daily, weekly, biweekly, monthly, quarterly, yearly, westmarch, one-shot} day:modal]
        - Re-schedule next session [/reschedule dropdown:campaign modal:next_date]
- New Session [/newsession dropdown:campaign]
    - Set date and time [if it is a westmarch, open modal to enter datetime, else, ask if it's a one-time edit or a complete reschedule (if it is, just open the /reschedule modal)]
    - Config [links to /configcampaign]
    - Description [modal text]
    - Links [link to links interaction]
    - Broadcast [button]
- Stats [/campaignstats dropdown:campaign]
    - Deaths [/addplayerdeath dropdown:campaign player]
    - Players joined
    - Players active [have played over the last month]
    - Players left [counter]
    - Campaign journal [/journal dropdown:campaign, can be set to off or hidden]

## Mod/Admin menu [/admin]

- Get active campaigns [/activecampaigns]
    - <selected_campaign> [this is what appears when a /managecampaign is used by a mod/admin]
        - DM [/getcampaigndm dropdown:campaign]
            - Ban from server [button, staff-only, link to /ban player reason]
        - Message Campaign DM [button, staff-only]
        - Ban from server [button, staff-only]
        - Stats [inherit from DM menu]
            - Times played
            - Players (status, playcount, deathcount)
            - Deaths total
            - Playtime (to be implemented, might use VC to track)
    - Get by DM [/campaigns format:dm]
    - Get All [/campaigns default]
- DMs [button, staff-only]
    - Ban from server [/ban player reason, staff-only]
    - Ban from making campaigns [/softban player reason, staff-only]
    - Unban member [/unban player reason, staff-only]
    - Campaigns [link to /campaigns format:dm player:dm-selector]
    - Stats [button, staff-only]
        - Campaignes owned
        - Campaigns playing/played
        - Time played
- Broadcast announcement [/announce message]

## Bot config [/settings]

- Database [button, staff-only]
    - List all Members
    - List all Campaigns
    - List Campaign-Player relationships
    - Stats
        - Active players
        - Active campaigns
        - Archived campaigns
        - Denied campaigns + DM
    - Audit log
- Add/remove permissions to user [button, staff-only]
    - [/grantpermission permission-id user]
- Debug
    - [add things here to see stats to debug issues]
    - [add copy of docker log]
- Storage used [button, staff-only]
- Enable/disable features [button, staff-only]
    - [buttons/interactions to allow/deny]
- About [/moontracer]

## Campaign Create [/newcampaign]

- Name
    - Tag is now a normalised version of the name
- Book
    - Dropdown with options: {5e, 5.5e, pathfinder, other common options, homebrew/other}
- Max players
    - If empty, it's an open campaign
- Synopsis and rules
- Game format
    - Dropdown: {campaign, westmarch, one-shot}

