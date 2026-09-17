package mq

// EventMessage 链上事件的 MQ 消息
type EventMessage struct {
	TxHash      string `json:"tx_hash"`
	LogIndex    uint   `json:"log_index"`
	BlockNumber uint64 `json:"block_number"`
	EventSig    string `json:"event_sig"`
	User        string `json:"user"`
	Amount      string `json:"amount"` // 或 reward
	RawData     string `json:"raw_data,omitempty"`
}
