package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StakeDAO struct{ db *gorm.DB }

func NewStakeDAO(db *gorm.DB) *StakeDAO { return &StakeDAO{db: db} }

func (d *StakeDAO) SaveStake(ctx context.Context, stake *Stake) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_address"}},
		DoUpdates: clause.AssignmentColumns([]string{"amount", "last_updated", "updated_at"}),
	}).Create(stake).Error
}

func (d *StakeDAO) GetStake(ctx context.Context, userAddress string) (*Stake, error) {
	var stake Stake
	err := d.db.WithContext(ctx).Where("user_address = ?", userAddress).First(&stake).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("stake not found")
		}
		return nil, err
	}
	return &stake, nil
}

func (d *StakeDAO) UpdateStake(ctx context.Context, stake *Stake) error {
	return d.db.WithContext(ctx).Save(stake).Error
}
