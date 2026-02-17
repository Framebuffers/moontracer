package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
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
