package dao

import (
	"context"

	"gorm.io/gorm"
)

type ClaimDAO struct{ db *gorm.DB }

func NewClaimDAO(db *gorm.DB) *ClaimDAO { return &ClaimDAO{db: db} }

func (d *ClaimDAO) SaveClaim(ctx context.Context, user, reward string) error {
	claim := &Claim{User: user, Reward: reward}
	return d.db.WithContext(ctx).Create(claim).Error
}

func (d *ClaimDAO) GetClaims(ctx context.Context, user string) ([]Claim, error) {
	var claims []Claim
	err := d.db.WithContext(ctx).Where("user = ?", user).Order("created_at DESC").Find(&claims).Error
	return claims, err
}
