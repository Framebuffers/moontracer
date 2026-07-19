package commands

/*
All returns every registered slash command.

How to add a new command:
 1. Create internal/commands/<name>.go with a struct that implements Command.
 2. Implement Data() and Execute(s, i).
 3. Add user-facing strings to internal/messages/messages.go — no inline literals.
 4. If elevation is needed, call auth.Authorize(db, userID, scope, campaignID) at the top of Execute.
    Available scopes: auth.ScopePlayer, auth.ScopeMod, auth.ScopeAdmin, auth.ScopeDM, auth.ScopeMember.
 5. Add the command to the slice below.
 6. Run go build ./... and go test ./... to verify.
*/
func All() []Command {
	return []Command{
		&aboutCommand{},
	}
}
