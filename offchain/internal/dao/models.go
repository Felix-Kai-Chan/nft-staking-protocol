package dao

import "time"

type Stake struct {
	ID          uint      `gorm:"primaryKey"`
	UserAddress string    `gorm:"column:user_address;type:varchar(42);uniqueIndex;not null"`
	Amount      string    `gorm:"column:amount;type:varchar(78);not null"`
	StakedAt    uint64    `gorm:"column:staked_at;not null"`
	RewardDebt  uint64    `gorm:"column:reward_debt;default:0"`
	LastUpdated uint64    `gorm:"column:last_updated;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Stake) TableName() string { return "stakes" }

type Claim struct {
	ID        uint      `gorm:"primaryKey"`
	User      string    `gorm:"column:user;type:varchar(42);index;not null"`
	Reward    string    `gorm:"column:reward;type:varchar(78);not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Claim) TableName() string { return "reward_claims" }

type SyncCursor struct {
	ID        uint   `gorm:"primaryKey"`
	LastBlock uint64 `gorm:"column:last_block;not null"`
	UpdatedAt int64  `gorm:"column:updated_at"`
}

func (SyncCursor) TableName() string { return "sync_cursor" }

type EventLog struct {
	ID          uint      `gorm:"primaryKey"`
	TxHash      string    `gorm:"column:tx_hash;type:varchar(66);uniqueIndex:uk_tx_log;not null"`
	LogIndex    uint      `gorm:"column:log_index;uniqueIndex:uk_tx_log;not null"`
	BlockNumber uint64    `gorm:"column:block_number;index;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (EventLog) TableName() string { return "event_log" }
