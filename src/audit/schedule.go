package audit

import (
	"fmt"
	"sort"

	"solsticedtl/src/amount"
	"solsticedtl/src/domain"
	"solsticedtl/src/policy"
)

// PaymentProjection estimates the next permitted tranche without mutating state.
type PaymentProjection struct {
	PaymentID    string        `json:"paymentId"`
	WindowID     string        `json:"windowId"`
	Asset        string        `json:"asset"`
	DueEpoch     int           `json:"dueEpoch"`
	Priority     int           `json:"priority"`
	Outstanding  amount.Amount `json:"outstanding"`
	TrancheLimit amount.Amount `json:"trancheLimit"`
	FeeEstimate  amount.Amount `json:"feeEstimate"`
	NetEstimate  amount.Amount `json:"netEstimate"`
	Status       string        `json:"status"`
}

func BuildSchedule(state *domain.PoolState, rules policy.Policy) ([]PaymentProjection, error) {
	out := make([]PaymentProjection, 0)
	for _, payment := range state.SortedPayments() {
		if !payment.IsOpen() {
			continue
		}
		outstanding := payment.Outstanding()
		limit, err := rules.MaxPartialFor(outstanding)
		if err != nil {
			return nil, fmt.Errorf("payment %s partial limit: %w", payment.ID, err)
		}
		fee, err := rules.SettlementFee(limit)
		if err != nil {
			return nil, fmt.Errorf("payment %s fee estimate: %w", payment.ID, err)
		}
		net, err := limit.Sub(fee)
		if err != nil {
			return nil, fmt.Errorf("payment %s net estimate: %w", payment.ID, err)
		}
		out = append(out, PaymentProjection{
			PaymentID: payment.ID, WindowID: payment.WindowID, Asset: payment.Asset,
			DueEpoch: payment.DueEpoch, Priority: payment.Priority,
			Outstanding: outstanding, TrancheLimit: limit,
			FeeEstimate: fee, NetEstimate: net, Status: string(payment.Status),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DueEpoch != out[j].DueEpoch {
			return out[i].DueEpoch < out[j].DueEpoch
		}
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].PaymentID < out[j].PaymentID
	})
	return out, nil
}
