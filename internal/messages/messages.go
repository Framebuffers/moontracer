package messages

/*
Messages:
  Every user-facing string and identifier lives here.

  Organization within each scope:
    - identifiers: command names, custom IDs, internal keys. Do not translate.
    - user-facing: labels, descriptions, error copy shown to Discord users.
*/

// Generic
const (
	// identifiers
	BotVersion = "v1.1.1b-nightly-mythril"
)

const (
	// user-facing
	GenericErrorMessage      = "🚫 Something went wrong."
	InvalidButtonDataMessage = "⚠️ Invalid button data."
)

// About (/about)
const (
	// identifiers
	AboutCommandName           = "about"
	AboutCommandGitHubRepoLink = "https://github.com/framebuffers/moontracer"
	AboutCommandWebsite        = "https://framebuffer.cl/moontracer"
	AboutCommandLicense        = "AGPL-v3.0"
)

const (
	// user-facing
	AboutCommandDesc         = "About this bot."
	AboutCommandGitHubLabel  = "💻 GitHub"
	AboutLabel               = "🐺 About"
	HelpLabel                = "❓ Help"
	AboutCommandBotDesc      = "...a D&D campaign manager for players, DM and spectators!"
	AboutCommandCopyright    = "(C) 2026 **Framebuffer**"
	AboutCommandHelp         = "Type `/help`, or press the **Help** button below for a list of commands."
	AboutCommandAwoo         = "awoo!"
	AboutCommandAttributions = "Thanks to the D&D r/Chile Discord server for giving me the idea, letting me test the bot on their server and give me feedback to improve this bot."
)

// Navigation button labels
const (
	// user-facing
	BackLabel      = ""
	HomeLabel      = "🏠"
	MyProfileLabel = HomeLabel
)
