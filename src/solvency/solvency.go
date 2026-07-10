package solvency

import (
	"sort"

	"solsticedtl/src/amount"
	"solsticedtl/src/domain"
)

type WindowView struct {
	WindowID           string
	Asset              string
	Reserve            amount.Amount
	Outstanding        amount.Amount
	AccruedFees        amount.Amount
	FreeClaims         amount.Amount
	ReservedClaims     amount.Amount
	PostCloseWithdrawn amount.Amount
	Margin             int64
	OpenPayments       int
	ClosedPayments     int
}

func BuildSnapshot(state *domain.PoolState, window *domain.Window) *domain.WindowSnapshot {
	snapshot := domain.NewWindowSnapshot(window.ID, state.Epoch)
	snapshot.Reserve = window.Reserve
	snapshot.AccruedFees = window.AccruedFees
	snapshot.SettlementVolume = window.SettlementVolume

	for _, paymentID := range window.Payments {
		payment := state.Payments[paymentID]
		if payment == nil {
			continue
		}
		outstanding := payment.Outstanding()
		if outstanding > 0 {
			snapshot.PaymentOutstanding[payment.ID] = outstanding
			snapshot.Outstanding = snapshot.Outstanding.MustAdd(outstanding)
			snapshot.OpenPaymentCount++
		} else {
			snapshot.ClosedPaymentCount++
		}
	}

	snapshot.AccountFree, snapshot.AccountReserved = windowAccountClaims(state, window)

	snapshot.SolvencyMargin = int64(snapshot.Reserve) - int64(snapshot.Outstanding) - int64(snapshot.AccountFree.Sum())
	return snapshot
}

func View(state *domain.PoolState, window *domain.Window) WindowView {
	open, closed := window.PaymentCount(state.Payments)
	freeClaims, reservedClaims := windowAccountClaims(state, window)
	view := WindowView{
		WindowID:           window.ID,
		Asset:              window.Asset,
		Reserve:            window.Reserve,
		Outstanding:        window.Outstanding(state.Payments),
		AccruedFees:        window.AccruedFees,
		PostCloseWithdrawn: window.PostCloseWithdrawn.Sum(),
		OpenPayments:       open,
		ClosedPayments:     closed,
	}
	view.FreeClaims = freeClaims.Sum()
	view.ReservedClaims = reservedClaims.Sum()
	view.Margin = int64(view.Reserve) - int64(view.Outstanding) - int64(view.FreeClaims)
	return view
}

func windowAccountClaims(state *domain.PoolState, window *domain.Window) (amount.Vector, amount.Vector) {
	free := amount.NewVector()
	reserved := amount.NewVector()

	for accountID, deposited := range window.DepositsByAccount {
		claim := deposited
		withdrawn := window.WithdrawnByAccount.Get(accountID)
		if withdrawn > 0 {
			claim = claim.MustSub(withdrawn)
		}
		if claim > 0 {
			free[accountID] = claim
		}
	}

	for _, paymentID := range window.Payments {
		payment := state.Payments[paymentID]
		if payment == nil || payment.Asset != window.Asset {
			continue
		}
		if payment.Amount > 0 {
			current := free.Get(payment.Source)
			if payment.Amount >= current {
				delete(free, payment.Source)
			} else {
				free[payment.Source] = current.MustSub(payment.Amount)
			}
		}
		if outstanding := payment.Outstanding(); outstanding > 0 {
			_ = reserved.Add(payment.Source, outstanding)
		}
	}

	return free, reserved
}

func SnapshotWithdrawalCapacity(snapshot *domain.WindowSnapshot, accountID string) amount.Amount {
	if snapshot == nil {
		return amount.Zero
	}
	return snapshot.AccountFree.Get(accountID)
}

func RemainingSnapshotCapacity(window *domain.Window, accountID string) amount.Amount {
	if window == nil || window.Snapshot == nil {
		return amount.Zero
	}
	capacity := SnapshotWithdrawalCapacity(window.Snapshot, accountID)
	used := window.PostCloseWithdrawn.Get(accountID)
	if used >= capacity {
		return amount.Zero
	}
	return capacity.MustSub(used)
}

func AssetIDs(state *domain.PoolState) []string {
	keys := make([]string, 0, len(state.Assets))
	for key := range state.Assets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func Reconciliation(state *domain.PoolState) []domain.ReconciliationLine {
	assets := AssetIDs(state)
	out := make([]domain.ReconciliationLine, 0, len(assets))
	for _, asset := range assets {
		line := domain.ReconciliationLine{
			Asset:          asset,
			Reserve:        state.TotalReserve(asset),
			FreeClaims:     state.TotalFreeClaims(asset),
			ReservedClaims: state.TotalReservedClaims(asset),
			Outstanding:    state.TotalOutstanding(asset),
		}
		for _, window := range state.Windows {
			if window.Asset != asset {
				continue
			}
			line.WindowCount++
			line.AccruedFees = line.AccruedFees.MustAdd(window.AccruedFees)
			line.PaymentCount += len(window.Payments)
			if window.Snapshot != nil {
				line.ClosedMargin += window.Snapshot.SolvencyMargin
			}
		}
		out = append(out, line)
	}
	return out
}

func CanClose(state *domain.PoolState, window *domain.Window) (bool, *domain.WindowSnapshot) {
	snapshot := BuildSnapshot(state, window)
	return snapshot.SolvencyMargin >= 0, snapshot
}

func SortedWindowViews(state *domain.PoolState) []WindowView {
	windows := state.SortedWindows()
	out := make([]WindowView, 0, len(windows))
	for _, window := range windows {
		out = append(out, View(state, window))
	}
	return out
}
