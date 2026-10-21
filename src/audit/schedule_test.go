package audit

import (
	"solsticedtl/src/amount"
	"solsticedtl/src/domain"
	"solsticedtl/src/policy"
	"testing"
)

func TestBuildSchedulePrioritizesDueEpochAndEstimatesNet(t *testing.T) {
	state := domain.NewPoolState("SolsticeDTL")
	first := domain.NewPayment("late", "window", "alice", "merchant", "usd", amount.Must(900), 1)
	first.DueEpoch = 4
	second := domain.NewPayment("early", "window", "alice", "merchant", "usd", amount.Must(200), 1)
	second.DueEpoch = 2
	state.Payments[first.ID], state.Payments[second.ID] = first, second
	result, err := BuildSchedule(state, policy.Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0].PaymentID != "early" {
		t.Fatalf("unexpected order: %+v", result)
	}
	if result[0].TrancheLimit != amount.Must(200) || result[0].FeeEstimate != amount.Must(6) || result[0].NetEstimate != amount.Must(194) {
		t.Fatalf("unexpected estimate: %+v", result[0])
	}
}
