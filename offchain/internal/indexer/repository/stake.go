package repository

import (
	"context"

	"gorm.io/gorm/clause" // ✅ 修正了导入路径
)

// SaveStake 保存或更新质押记录 (Upsert 逻辑)
func (r *Repository) SaveStake(ctx context.Context, stake *Stake) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_address"}},                          // 冲突的列
		DoUpdates: clause.AssignmentColumns([]string{"last_updated", "updated_at"}), // ✅ 移除了 'token_id'
	}).Create(stake).Error
}

// GetStake 获取质押记录
func (r *Repository) GetStake(ctx context.Context, userAddress string) (*Stake, error) {
	var stake Stake
	err := r.db.WithContext(ctx).
		Where("user_address = ?", userAddress).
		First(&stake).Error
	if err != nil {
		return nil, err
	}
	return &stake, nil
}

// UpdateStake 更新质押记录
func (r *Repository) UpdateStake(ctx context.Context, stake *Stake) error {
	return r.db.WithContext(ctx).Save(stake).Error
}

// DeleteStake 删除质押记录（取回后删除）
func (r *Repository) DeleteStake(ctx context.Context, userAddress string) error {
	return r.db.WithContext(ctx).
		Where("user_address = ?", userAddress).
		Delete(&Stake{}).Error
}

// SaveClaim 保存领取记录
func (r *Repository) SaveClaim(ctx context.Context, user, reward string) error {
	claim := &Claim{
		User:   user,
		Reward: reward,
	}
	return r.db.WithContext(ctx).Create(claim).Error
}

// GetClaims 查询用户领取历史
func (r *Repository) GetClaims(ctx context.Context, user string) ([]Claim, error) {
	var claims []Claim
	err := r.db.WithContext(ctx).
		Where("user = ?", user).
		Order("created_at DESC").
		Find(&claims).Error
	return claims, err
}
