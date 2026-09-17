package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type EventLogDAO struct{ db *gorm.DB }

func NewEventLogDAO(db *gorm.DB) *EventLogDAO { return &EventLogDAO{db: db} }

var ErrDuplicate = errors.New("duplicate event")

func (d *EventLogDAO) Insert(ctx context.Context, txHash string, logIndex uint, blockNumber uint64) error {
	event := &EventLog{TxHash: txHash, LogIndex: logIndex, BlockNumber: blockNumber}
	err := d.db.WithContext(ctx).Create(event).Error
	if err != nil {
		// GORM v1.25+ 支持 errors.Is(err, gorm.ErrDuplicatedKey)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}
