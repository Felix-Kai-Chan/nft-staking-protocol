package mq

import (
	"context"
	"testing"
	"time"

	"staking-offchain/internal/dao"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testMQURL = "amqp://guest:guest@localhost:5672/"
	testQueue = "staking_events_test"
	testDSN   = "root:@tcp(127.0.0.1:3306)/staking_test?charset=utf8mb4&parseTime=True&loc=Local"
)

// setupTestDB 初始化测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(mysql.Open(testDSN), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Skipf("MySQL 不可用，跳过测试: %v", err)
	}

	if err := db.AutoMigrate(&dao.Stake{}, &dao.Claim{}, &dao.SyncCursor{}, &dao.EventLog{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}

	db.Exec("TRUNCATE TABLE stakes")
	db.Exec("TRUNCATE TABLE reward_claims")
	db.Exec("TRUNCATE TABLE sync_cursor")
	db.Exec("TRUNCATE TABLE event_log")

	return db
}

// setupTestMQ 初始化 MQ 连接
func setupTestMQ(t *testing.T) (*Producer, *Consumer, *gorm.DB) {
	db := setupTestDB(t)

	producer, err := NewProducer(testMQURL, testQueue)
	if err != nil {
		t.Skipf("RabbitMQ 不可用，跳过测试: %v", err)
	}

	consumer, err := NewConsumer(testMQURL, testQueue, db)
	if err != nil {
		producer.Close()
		t.Skipf("RabbitMQ Consumer 创建失败，跳过测试: %v", err)
	}

	return producer, consumer, db
}

// TestProducer_Publish 测试 Producer 投递消息
func TestProducer_Publish(t *testing.T) {
	producer, _, _ := setupTestMQ(t)
	defer producer.Close()

	ctx := context.Background()
	msg := &EventMessage{
		TxHash:      "0xabc123",
		LogIndex:    0,
		BlockNumber: 100,
		EventSig:    "Staked",
		User:        "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		Amount:      "1000000000000000000000",
	}

	if err := producer.Publish(ctx, msg); err != nil {
		t.Fatalf("Publish 失败: %v", err)
	}

	t.Logf("✅ Producer 通过：投递成功 tx=%s", msg.TxHash)
}

// TestConsumer_Consume 测试 Consumer 消费消息 + 写 DAO
func TestConsumer_Consume(t *testing.T) {
	producer, consumer, db := setupTestMQ(t)
	defer producer.Close()
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动 Consumer（goroutine）
	go consumer.Start(ctx)

	// 等 Consumer 就绪
	time.Sleep(500 * time.Millisecond)

	// 投递消息
	msg := &EventMessage{
		TxHash:      "0xconsumer_test_001",
		LogIndex:    0,
		BlockNumber: 200,
		EventSig:    "Staked",
		User:        "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		Amount:      "1000000000000000000000",
	}
	if err := producer.Publish(ctx, msg); err != nil {
		t.Fatalf("Publish 失败: %v", err)
	}

	// 等 Consumer 消费
	time.Sleep(1 * time.Second)

	// 验证 event_log 有记录
	var count int64
	db.Model(&dao.EventLog{}).Where("tx_hash = ?", msg.TxHash).Count(&count)
	if count != 1 {
		t.Errorf("event_log 期望 1 条，实际 %d", count)
	}

	// 验证 stakes 有记录
	var stakeCount int64
	db.Model(&dao.Stake{}).Where("user_address = ?", msg.User).Count(&stakeCount)
	if stakeCount != 1 {
		t.Errorf("stakes 期望 1 条，实际 %d", stakeCount)
	}

	t.Logf("✅ Consumer 通过：消费成功，event_log=%d, stakes=%d", count, stakeCount)
}

// TestConsumer_Duplicate 测试重复消费被去重表拦截
func TestConsumer_Duplicate(t *testing.T) {
	producer, consumer, db := setupTestMQ(t)
	defer producer.Close()
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Start(ctx)
	time.Sleep(500 * time.Millisecond)

	// 投递同一条消息 2 次
	msg := &EventMessage{
		TxHash:      "0xduplicate_test_001",
		LogIndex:    0,
		BlockNumber: 300,
		EventSig:    "Staked",
		User:        "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		Amount:      "1000",
	}

	// 第一次
	if err := producer.Publish(ctx, msg); err != nil {
		t.Fatalf("第一次 Publish 失败: %v", err)
	}
	time.Sleep(800 * time.Millisecond)

	// 第二次（同 tx_hash + log_index）
	if err := producer.Publish(ctx, msg); err != nil {
		t.Fatalf("第二次 Publish 失败: %v", err)
	}
	time.Sleep(800 * time.Millisecond)

	// 验证 event_log 只有 1 条
	var count int64
	db.Model(&dao.EventLog{}).Where("tx_hash = ?", msg.TxHash).Count(&count)
	if count != 1 {
		t.Errorf("event_log 期望 1 条（幂等），实际 %d", count)
	}

	// 验证 stakes 只有 1 条
	var stakeCount int64
	db.Model(&dao.Stake{}).Where("user_address = ?", msg.User).Count(&stakeCount)
	if stakeCount != 1 {
		t.Errorf("stakes 期望 1 条（Upsert），实际 %d", stakeCount)
	}

	t.Logf("✅ Duplicate 通过：event_log=%d, stakes=%d", count, stakeCount)
}

// TestConsumer_Claimed 测试 Claimed 事件
func TestConsumer_Claimed(t *testing.T) {
	producer, consumer, db := setupTestMQ(t)
	defer producer.Close()
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Start(ctx)
	time.Sleep(500 * time.Millisecond)

	msg := &EventMessage{
		TxHash:      "0xclaimed_test_001",
		LogIndex:    0,
		BlockNumber: 400,
		EventSig:    "Claimed",
		User:        "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		Amount:      "317000000000000000000",
	}
	if err := producer.Publish(ctx, msg); err != nil {
		t.Fatalf("Publish 失败: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 验证 reward_claims 有记录
	var count int64
	db.Model(&dao.Claim{}).Where("user = ?", msg.User).Count(&count)
	if count != 1 {
		t.Errorf("reward_claims 期望 1 条，实际 %d", count)
	}

	t.Logf("✅ Claimed 通过：reward_claims=%d", count)
}
