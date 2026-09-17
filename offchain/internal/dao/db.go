package dao

import (
	"log/slog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接 + AutoMigrate
func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(500)
	sqlDB.SetMaxIdleConns(100)

	if err := db.AutoMigrate(
		&Stake{},
		&Claim{},
		&SyncCursor{},
		&EventLog{},
	); err != nil {
		return nil, err
	}

	slog.Info("DAO: database migrated successfully")
	DB = db
	return db, nil
}
