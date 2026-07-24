package dto

// StakeRequest 质押请求
type StakeRequest struct {
	Amount string `json:"amount" binding:"required"` // 字符串，防止 uint256 溢出
}

// WithdrawRequest 提现请求
type WithdrawRequest struct {
	Amount string `json:"amount" binding:"required"`
}

// ClaimRequest 领取奖励请求（无参数）
type ClaimRequest struct {
	// 空
}

// StakeResponse 质押响应
type StakeResponse struct {
	TxHash string `json:"tx_hash"`
	Amount string `json:"amount"`
}

// WithdrawResponse 提现响应
type WithdrawResponse struct {
	TxHash string `json:"tx_hash"`
	Amount string `json:"amount"`
}

// ClaimResponse 领取奖励响应
type ClaimResponse struct {
	TxHash string `json:"tx_hash"`
}
