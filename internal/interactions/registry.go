package interactions

import (
	"github.com/uptrace/bun"
)

func AllComponents(db *bun.DB, guildID string, adminRoleName string) []ComponentHandler {
	return []ComponentHandler{
		&campaignJoin{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&campaignLeave{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&campaignToggle{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&campaignView{db: db, guildID: guildID, adminRoleName: adminRoleName},
	}
}

func AllModals(db *bun.DB, guildID, adminRole string) []ModalHandler {
	return []ModalHandler{
		&modalCampaignCreate{db: db},
	}
}
