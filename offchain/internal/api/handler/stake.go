package handler

import (
	"net/http"

	"staking-offchain/internal/api/dto"
	"staking-offchain/internal/api/service"

	"github.com/gin-gonic/gin"
)

type StakeHandler struct {
	stakeService *service.StakeService
}

func NewStakeHandler(stakeService *service.StakeService) *StakeHandler {
	return &StakeHandler{stakeService: stakeService}
}

// GetStakeInfo 查询用户质押信息
func (h *StakeHandler) GetStakeInfo(c *gin.Context) {
	address := c.Param("address")

	stake, err := h.stakeService.GetStakeInfo(c.Request.Context(), address)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no stake found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"address":   stake.UserAddress,
		"amount":    stake.Amount,
		"staked_at": stake.StakedAt,
	})
}

// GetClaims 查询用户领取历史
func (h *StakeHandler) GetClaims(c *gin.Context) {
	address := c.Param("address")

	claims, err := h.stakeService.GetClaims(c.Request.Context(), address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"claims":  claims,
	})
}

// Stake 质押
func (h *StakeHandler) Stake(c *gin.Context) {
	var req dto.StakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txHash, err := h.stakeService.Stake(c.Request.Context(), req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.StakeResponse{
		TxHash: txHash,
		Amount: req.Amount,
	})
}

// Withdraw 提现
func (h *StakeHandler) Withdraw(c *gin.Context) {
	var req dto.WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txHash, err := h.stakeService.Withdraw(c.Request.Context(), req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.WithdrawResponse{
		TxHash: txHash,
		Amount: req.Amount,
	})
}

// Claim 领取奖励
func (h *StakeHandler) Claim(c *gin.Context) {
	txHash, err := h.stakeService.Claim(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ClaimResponse{
		TxHash: txHash,
	})
}
