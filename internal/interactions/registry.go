package interactions

import (
	"github.com/uptrace/bun"
)

func AllComponents(db *bun.DB, guildID, adminRole string) []ComponentHandler {
	return []ComponentHandler{
		&campaignJoin{db: db},
		&campaignLeave{db: db},
		&campaignToggle{db: db},
		&campaignView{db: db},
	}
}

func AllModals(db *bun.DB, guildID, adminRole string) []ModalHandler {
	return []ModalHandler{
		&modalCampaignCreate{db: db},
	}
}
