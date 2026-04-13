package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
)

func GetByID[T any](db *bun.DB, id string) (*T, error) {
	ctx := context.Background()
	var model T
	err := db.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func GetByTag[T any](db *bun.DB, tag string) (*T, error) {
	ctx := context.Background()
	var model T
	err := db.NewSelect().Model(&model).Where("tag = ?", tag).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func GetAll[T any](db *bun.DB) ([]T, error) {
	ctx := context.Background()
	var models []T
	err := db.NewSelect().Model(&models).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return models, nil
}

func GetStaff(db *bun.DB) ([]models.Player, error) {
	ctx := context.Background()
	var players []models.Player
	err := db.NewSelect().
		Model(&players).
		Where("role IN (?)", bun.In([]models.ServerRole{models.ServerRoleMod, models.ServerRoleAdmin})).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return players, nil
}

func Update[T any](db *bun.DB, model *T) error {
	ctx := context.Background()

	exists, err := db.NewSelect().Model(model).WherePK().Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("record not found")
	}

	_, err = db.NewUpdate().Model(model).WherePK().Exec(ctx)
	return err
}

func Delete[T any](db *bun.DB, id string) error {
	ctx := context.Background()
	_, err := db.NewDelete().Model((*T)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func Insert[T any](db *bun.DB, model *T) error {
	ctx := context.Background()
	_, err := db.NewInsert().Model(model).Exec(ctx)
	return err
}

/*
ScrubOrphanedCampaignPlayers removes CampaignPlayer rows whose campaign
no longer exists. This cleans up records left by older code that deleted
campaigns without cascade-deleting their players.
*/
func ScrubOrphanedCampaignPlayers(db *bun.DB) (int64, error) {
	ctx := context.Background()
	res, err := db.NewDelete().
		Model((*models.CampaignPlayer)(nil)).
		Where("campaign_id NOT IN (?)",
			db.NewSelect().Model((*models.Campaign)(nil)).Column("id"),
		).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
