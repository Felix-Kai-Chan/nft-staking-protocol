package dao

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB 初始化测试数据库
// 需要本地 MySQL 在 3306，数据库 staking_test
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "root:@tcp(127.0.0.1:3306)/staking_test?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Skipf("MySQL 不可用，跳过测试: %v", err)
	}

	// AutoMigrate
	if err := db.AutoMigrate(&Stake{}, &Claim{}, &SyncCursor{}, &EventLog{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}

	// 清空表
	db.Exec("TRUNCATE TABLE stakes")
	db.Exec("TRUNCATE TABLE reward_claims")
	db.Exec("TRUNCATE TABLE sync_cursor")
	db.Exec("TRUNCATE TABLE event_log")

	return db
}

// TestStakeDAO_SaveAndGet 测试保存 + 查询
func TestStakeDAO_SaveAndGet(t *testing.T) {
	db := setupTestDB(t)
	stakeDAO := NewStakeDAO(db)
	ctx := context.Background()

	stake := &Stake{
		UserAddress: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		Amount:      "1000000000000000000000",
		StakedAt:    uint64(time.Now().Unix()),
		LastUpdated: uint64(time.Now().Unix()),
	}

	// 保存
	if err := stakeDAO.SaveStake(ctx, stake); err != nil {
		t.Fatalf("SaveStake 失败: %v", err)
	}

	// 查询
	got, err := stakeDAO.GetStake(ctx, stake.UserAddress)
	if err != nil {
		t.Fatalf("GetStake 失败: %v", err)
	}

	if got.UserAddress != stake.UserAddress {
		t.Errorf("UserAddress 不匹配: got %s, want %s", got.UserAddress, stake.UserAddress)
	}
	if got.Amount != stake.Amount {
		t.Errorf("Amount 不匹配: got %s, want %s", got.Amount, stake.Amount)
	}

	t.Logf("✅ SaveAndGet 通过：user=%s, amount=%s", got.UserAddress, got.Amount)
}

// TestStakeDAO_Upsert 测试 Upsert（不重复插入）
func TestStakeDAO_Upsert(t *testing.T) {
	db := setupTestDB(t)
	stakeDAO := NewStakeDAO(db)
	ctx := context.Background()

	user := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	// 第一次插入
	stake1 := &Stake{
		UserAddress: user,
		Amount:      "1000000000000000000000",
		StakedAt:    1,
		LastUpdated: 1,
	}
	if err := stakeDAO.SaveStake(ctx, stake1); err != nil {
		t.Fatalf("第一次 SaveStake 失败: %v", err)
	}

	// 第二次插入（同一个 user，应该 update）
	stake2 := &Stake{
		UserAddress: user,
		Amount:      "2000000000000000000000",
		StakedAt:    1,
		LastUpdated: 2,
	}
	if err := stakeDAO.SaveStake(ctx, stake2); err != nil {
		t.Fatalf("第二次 SaveStake 失败: %v", err)
	}

	// 查询，应该只有 1 条记录
	var count int64
	db.Model(&Stake{}).Where("user_address = ?", user).Count(&count)
	if count != 1 {
		t.Errorf("Upsert 失败：期望 1 条记录，实际 %d", count)
	}

	// 金额应该是最新的
	got, _ := stakeDAO.GetStake(ctx, user)
	if got.Amount != "2000000000000000000000" {
		t.Errorf("Amount 未更新: got %s", got.Amount)
	}

	t.Logf("✅ Upsert 通过：只有 1 条记录，金额已更新")
}

// TestEventLogDAO_Duplicate 测试去重表拦截重复
func TestEventLogDAO_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	eventLogDAO := NewEventLogDAO(db)
	ctx := context.Background()

	txHash := "0xa80f8c2a91e138e635ee7e54e6b19526540f359487615836a53d3d39cb0a8c9c"
	logIndex := uint(1)
	blockNumber := uint64(16)

	// 第一次插入 → 成功
	if err := eventLogDAO.Insert(ctx, txHash, logIndex, blockNumber); err != nil {
		t.Fatalf("第一次 Insert 失败: %v", err)
	}

	// 第二次插入（同一 tx_hash + log_index）→ 应该返回 ErrDuplicate
	err := eventLogDAO.Insert(ctx, txHash, logIndex, blockNumber)
	if err != ErrDuplicate {
		t.Errorf("期望 ErrDuplicate，实际: %v", err)
	}

	t.Logf("✅ 去重表通过：第二次插入返回 ErrDuplicate")
}

// TestCursorDAO_GetAndUpdate 测试 Cursor 读写
func TestCursorDAO_GetAndUpdate(t *testing.T) {
	db := setupTestDB(t)
	cursorDAO := NewCursorDAO(db)
	ctx := context.Background()

	// 初始 → 0
	block, err := cursorDAO.GetLastBlock(ctx)
	if err != nil {
		t.Fatalf("GetLastBlock 失败: %v", err)
	}
	if block != 0 {
		t.Errorf("初始 block 期望 0，实际 %d", block)
	}

	// 更新到 100
	if err := cursorDAO.UpdateLastBlock(ctx, 100); err != nil {
		t.Fatalf("UpdateLastBlock 失败: %v", err)
	}

	// 再查 → 100
	block, _ = cursorDAO.GetLastBlock(ctx)
	if block != 100 {
		t.Errorf("block 期望 100，实际 %d", block)
	}

	t.Logf("✅ Cursor 通过：初始 0，更新到 100")
}

// TestClaimDAO_SaveAndGet 测试 Claim 保存 + 查询
func TestClaimDAO_SaveAndGet(t *testing.T) {
	db := setupTestDB(t)
	claimDAO := NewClaimDAO(db)
	ctx := context.Background()

	user := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	// 保存 2 条记录
	if err := claimDAO.SaveClaim(ctx, user, "100"); err != nil {
		t.Fatalf("SaveClaim 失败: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // 确保时间戳不同
	if err := claimDAO.SaveClaim(ctx, user, "200"); err != nil {
		t.Fatalf("SaveClaim 失败: %v", err)
	}

	// 查询
	claims, err := claimDAO.GetClaims(ctx, user)
	if err != nil {
		t.Fatalf("GetClaims 失败: %v", err)
	}

	if len(claims) != 2 {
		t.Errorf("期望 2 条记录，实际 %d", len(claims))
	}

	// 按时间倒序：最新的 200 在前
	if claims[0].Reward != "200" {
		t.Errorf("第一条应为 200，实际 %s", claims[0].Reward)
	}

	t.Logf("✅ Claim 通过：2 条记录，按时间倒序")
}

// TestMain 设置测试环境
func TestMain(m *testing.M) {
	// 检查环境变量
	if os.Getenv("SKIP_DB_TEST") == "1" {
		fmt.Println("SKIP_DB_TEST=1，跳过所有测试")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
