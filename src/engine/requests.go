package engine

import "solsticedtl/src/amount"

type AccountRequest struct {
	ID          string            `json:"id"`
	Role        string            `json:"role"`
	DisplayName string            `json:"displayName"`
	Metadata    map[string]string `json:"metadata"`
}

type AssetRequest struct {
	ID                 string            `json:"id"`
	Symbol             string            `json:"symbol"`
	Decimals           int               `json:"decimals"`
	SettlementPriority int               `json:"settlementPriority"`
	Metadata           map[string]string `json:"metadata"`
}

type WindowRequest struct {
	ID         string            `json:"id"`
	Asset      string            `json:"asset"`
	Label      string            `json:"label"`
	OpenEpoch  int               `json:"openEpoch"`
	CloseEpoch int               `json:"closeEpoch"`
	Metadata   map[string]string `json:"metadata"`
}

type DepositRequest struct {
	AccountID string        `json:"accountId"`
	WindowID  string        `json:"windowId"`
	Asset     string        `json:"asset"`
	Amount    amount.Amount `json:"amount"`
}

type QueuePaymentRequest struct {
	ID          string            `json:"id"`
	WindowID    string            `json:"windowId"`
	Source      string            `json:"source"`
	Beneficiary string            `json:"beneficiary"`
	Asset       string            `json:"asset"`
	Amount      amount.Amount     `json:"amount"`
	DueEpoch    int               `json:"dueEpoch"`
	Priority    int               `json:"priority"`
	Metadata    map[string]string `json:"metadata"`
}

type CloseWindowRequest struct {
	WindowID string `json:"windowId"`
}

type SettlementRequest struct {
	PaymentID string        `json:"paymentId"`
	WindowID  string        `json:"windowId"`
	Amount    amount.Amount `json:"amount"`
	Operator  string        `json:"operator"`
}

type WithdrawalRequest struct {
	AccountID string        `json:"accountId"`
	WindowID  string        `json:"windowId"`
	Asset     string        `json:"asset"`
	Amount    amount.Amount `json:"amount"`
}

type AdvanceRequest struct {
	Epochs int `json:"epochs"`
	To     int `json:"to"`
}
