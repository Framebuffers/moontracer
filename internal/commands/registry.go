package commands

// All returns every registered command. Add new commands here.
func All() []Command {
	return []Command{
		&pingCommand{},
		&awooCommand{},
		&campaignCommand{},
	}
}
