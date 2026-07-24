package repository

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Repository 负责数据存储
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建 Repository 实例
func NewRepository(dsn string) (*Repository, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移：创建或更新表结构
	if err := db.AutoMigrate(&Stake{}, &Claim{}, &SyncStatus{}); err != nil {
		return nil, err
	}

	log.Println("✅ Database tables migrated successfully")
	return &Repository{db: db}, nil
}

// NewRepositoryFromDB 从已有 *gorm.DB 创建 Repository（用于 API）
func NewRepositoryFromDB(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetDB 获取 gorm.DB 实例
func (r *Repository) GetDB() *gorm.DB {
	return r.db
}

// ========== Models (模型定义) ==========

// Stake 质押记录 (已适配 ERC20 质押)
type Stake struct {
	ID          uint      `gorm:"primaryKey"`
	UserAddress string    `gorm:"column:user_address;type:varchar(42);uniqueIndex;not null"`
	Amount      string    `gorm:"column:amount;type:varchar(78);not null"` // 质押金额 (使用 string 防止 uint256 溢出)
	StakedAt    uint64    `gorm:"column:staked_at;not null"`               // 质押时间戳
	RewardDebt  uint64    `gorm:"column:reward_debt;default:0"`            // 已结算奖励
	LastUpdated uint64    `gorm:"column:last_updated;not null"`            // 上次更新时间
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Stake) TableName() string {
	return "stakes"
}

// Claim 领取记录
type Claim struct {
	ID        uint      `gorm:"primaryKey"`
	User      string    `gorm:"column:user;type:varchar(42);index;not null"`
	Reward    string    `gorm:"column:reward;type:varchar(78);not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Claim) TableName() string {
	return "reward_claims"
}

// SyncStatus 同步状态记录
type SyncStatus struct {
	ID        uint   `gorm:"primaryKey"`
	LastBlock uint64 `gorm:"not null"` // 上次成功同步的区块号
	UpdatedAt int64  // 更新时间戳
}

func (SyncStatus) TableName() string {
	return "sync_status"
}
