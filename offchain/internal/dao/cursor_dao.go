package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type CursorDAO struct{ db *gorm.DB }

func NewCursorDAO(db *gorm.DB) *CursorDAO { return &CursorDAO{db: db} }

func (d *CursorDAO) GetLastBlock(ctx context.Context) (uint64, error) {
	var cursor SyncCursor
	err := d.db.WithContext(ctx).Where("id = ?", 1).First(&cursor).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return cursor.LastBlock, nil
}

func (d *CursorDAO) UpdateLastBlock(ctx context.Context, blockNumber uint64) error {
	cursor := SyncCursor{ID: 1, LastBlock: blockNumber, UpdatedAt: time.Now().Unix()}
	return d.db.WithContext(ctx).Save(&cursor).Error
}
