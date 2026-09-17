package dao

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupBenchDB 初始化 Bench 数据库
func setupBenchDB(b *testing.B) *gorm.DB {
	dsn := "root:@tcp(127.0.0.1:3306)/staking_test?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		b.Skipf("MySQL 不可用，跳过 benchmark: %v", err)
	}

	if err := db.AutoMigrate(&Stake{}, &Claim{}, &SyncCursor{}, &EventLog{}); err != nil {
		b.Fatalf("AutoMigrate 失败: %v", err)
	}

	db.Exec("TRUNCATE TABLE stakes")
	db.Exec("TRUNCATE TABLE event_log")
	db.Exec("TRUNCATE TABLE sync_cursor")

	return db
}

// BenchmarkStakeDAO_Save 测试 SaveStake 性能
func BenchmarkStakeDAO_Save(b *testing.B) {
	db := setupBenchDB(b)
	stakeDAO := NewStakeDAO(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stake := &Stake{
			UserAddress: fmt.Sprintf("0xuser_%d", i),
			Amount:      "1000000000000000000000",
			StakedAt:    uint64(time.Now().Unix()),
			LastUpdated: uint64(time.Now().Unix()),
		}
		if err := stakeDAO.SaveStake(ctx, stake); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStakeDAO_Get 测试 GetStake 性能
func BenchmarkStakeDAO_Get(b *testing.B) {
	db := setupBenchDB(b)
	stakeDAO := NewStakeDAO(db)
	ctx := context.Background()

	// 预置数据
	user := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	stakeDAO.SaveStake(ctx, &Stake{
		UserAddress: user,
		Amount:      "1000",
		StakedAt:    1,
		LastUpdated: 1,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := stakeDAO.GetStake(ctx, user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventLogInsert 测试去重表插入性能
func BenchmarkEventLogInsert(b *testing.B) {
	db := setupBenchDB(b)
	eventLogDAO := NewEventLogDAO(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txHash := fmt.Sprintf("0x%064d", i)
		err := eventLogDAO.Insert(ctx, txHash, 0, uint64(i))
		if err != nil && err != ErrDuplicate {
			b.Fatal(err)
		}
	}
}

// BenchmarkCursorGet 测试 Cursor 读取性能
func BenchmarkCursorGet(b *testing.B) {
	db := setupBenchDB(b)
	cursorDAO := NewCursorDAO(db)
	ctx := context.Background()

	cursorDAO.UpdateLastBlock(ctx, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cursorDAO.GetLastBlock(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStakeDAO_Upsert 测试 Upsert 性能
func BenchmarkStakeDAO_Upsert(b *testing.B) {
	db := setupBenchDB(b)
	stakeDAO := NewStakeDAO(db)
	ctx := context.Background()

	user := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stake := &Stake{
			UserAddress: user,
			Amount:      fmt.Sprintf("%d", i),
			StakedAt:    1,
			LastUpdated: uint64(i),
		}
		if err := stakeDAO.SaveStake(ctx, stake); err != nil {
			b.Fatal(err)
		}
	}
}
