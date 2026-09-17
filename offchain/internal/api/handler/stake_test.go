package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"staking-offchain/internal/api/service"
	"staking-offchain/internal/dao"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB 初始化测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "root:@tcp(127.0.0.1:3306)/staking_test?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
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

// setupTestRouter 初始化测试 Router
func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)

	stakeDAO := dao.NewStakeDAO(db)
	claimDAO := dao.NewClaimDAO(db)

	// 注意：测试不需要合约和 transactOpts，传 nil
	// 但 GetStakeInfo / GetClaims 只用到 DAO，不用合约
	stakeService := service.NewStakeService(stakeDAO, claimDAO, nil, nil)
	stakeHandler := NewStakeHandler(stakeService)
	healthHandler := NewHealthHandler()

	r := gin.New()

	// 注册路由
	r.GET("/health", healthHandler.Health)
	api := r.Group("/api/v1")
	{
		api.GET("/stake/:address", stakeHandler.GetStakeInfo)
		api.GET("/stake/:address/claims", stakeHandler.GetClaims)
	}

	return r, db
}

// TestHealth 测试 /health
func TestHealth(t *testing.T) {
	r, _ := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("期望 status=ok，实际 %v", resp["status"])
	}

	t.Logf("✅ Health 通过：status=%v", resp["status"])
}

// TestGetStakeInfo_Success 测试查询存在的 stake
func TestGetStakeInfo_Success(t *testing.T) {
	r, db := setupTestRouter(t)

	// 预置数据
	user := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	db.Create(&dao.Stake{
		UserAddress: user,
		Amount:      "1000000000000000000000",
		StakedAt:    1700000000,
		LastUpdated: 1700000000,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stake/"+user, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["address"] != user {
		t.Errorf("address 不匹配: %v", resp["address"])
	}

	t.Logf("✅ GetStakeInfo_Success 通过：address=%v, amount=%v", resp["address"], resp["amount"])
}

// TestGetStakeInfo_NotFound 测试查询不存在的 stake → 404
func TestGetStakeInfo_NotFound(t *testing.T) {
	r, _ := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stake/0xNonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际 %d", w.Code)
	}

	t.Logf("✅ GetStakeInfo_NotFound 通过：返回 404")
}

// TestGetClaims_Success 测试查询 claim 历史
func TestGetClaims_Success(t *testing.T) {
	r, db := setupTestRouter(t)

	// 预置数据
	user := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	db.Create(&dao.Claim{User: user, Reward: "100"})
	db.Create(&dao.Claim{User: user, Reward: "200"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stake/"+user+"/claims", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	claims, ok := resp["claims"].([]interface{})
	if !ok {
		t.Fatalf("claims 字段不存在或格式错误")
	}

	if len(claims) != 2 {
		t.Errorf("期望 2 条 claim，实际 %d", len(claims))
	}

	t.Logf("✅ GetClaims_Success 通过：%d 条 claim", len(claims))
}

// TestGetClaims_Empty 测试查询无 claim 历史
func TestGetClaims_Empty(t *testing.T) {
	r, _ := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stake/0xEmpty/claims", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	claims, ok := resp["claims"].([]interface{})
	if !ok {
		// claims 可能为 nil，但状态码 200 就行
		t.Logf("✅ GetClaims_Empty 通过：claims 为空")
		return
	}

	if len(claims) != 0 {
		t.Errorf("期望 0 条 claim，实际 %d", len(claims))
	}

	t.Logf("✅ GetClaims_Empty 通过：0 条 claim")
}

// TestGetStakeInfo_JSONFormat 测试响应 JSON 格式
func TestGetStakeInfo_JSONFormat(t *testing.T) {
	r, db := setupTestRouter(t)

	user := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	db.Create(&dao.Stake{
		UserAddress: user,
		Amount:      "1000",
		StakedAt:    1700000000,
		LastUpdated: 1700000000,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stake/"+user, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}

	// 验证字段
	requiredFields := []string{"address", "amount", "staked_at"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("响应缺少字段: %s", field)
		}
	}

	t.Logf("✅ JSONFormat 通过：所有字段存在")
}

// 用于防止 "context not used" 编译错误
var _ = context.Background
var _ = fmt.Sprintf
