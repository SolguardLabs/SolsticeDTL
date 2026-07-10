package domain

import "solsticedtl/src/amount"

type AmountEntry struct {
	ID     string        `json:"id"`
	Amount amount.Amount `json:"amount"`
}

type AssetReport struct {
	ID                 string `json:"id"`
	Symbol             string `json:"symbol"`
	Decimals           int    `json:"decimals"`
	SettlementPriority int    `json:"settlementPriority"`
	Active             bool   `json:"active"`
}

type AccountReport struct {
	ID          string        `json:"id"`
	Role        AccountRole   `json:"role"`
	Free        []AmountEntry `json:"free"`
	Reserved    []AmountEntry `json:"reserved"`
	SettledIn   []AmountEntry `json:"settledIn"`
	Withdrawn   []AmountEntry `json:"withdrawn"`
	FeesPaid    []AmountEntry `json:"feesPaid"`
	TotalClaims []AmountEntry `json:"totalClaims"`
}

type SnapshotReport struct {
	Epoch              int           `json:"epoch"`
	Reserve            amount.Amount `json:"reserve"`
	Outstanding        amount.Amount `json:"outstanding"`
	AccruedFees        amount.Amount `json:"accruedFees"`
	SettlementVolume   amount.Amount `json:"settlementVolume"`
	SolvencyMargin     int64         `json:"solvencyMargin"`
	ClosedPaymentCount int           `json:"closedPaymentCount"`
	OpenPaymentCount   int           `json:"openPaymentCount"`
	AccountFree        []AmountEntry `json:"accountFree"`
	AccountReserved    []AmountEntry `json:"accountReserved"`
}

type WindowReport struct {
	ID                 string          `json:"id"`
	Asset              string          `json:"asset"`
	Status             WindowStatus    `json:"status"`
	OpenEpoch          int             `json:"openEpoch"`
	CloseEpoch         int             `json:"closeEpoch"`
	Reserve            amount.Amount   `json:"reserve"`
	GrossDeposits      amount.Amount   `json:"grossDeposits"`
	GrossWithdrawals   amount.Amount   `json:"grossWithdrawals"`
	SettlementVolume   amount.Amount   `json:"settlementVolume"`
	AccruedFees        amount.Amount   `json:"accruedFees"`
	Outstanding        amount.Amount   `json:"outstanding"`
	PaymentCount       int             `json:"paymentCount"`
	PostCloseWithdrawn []AmountEntry   `json:"postCloseWithdrawn"`
	Snapshot           *SnapshotReport `json:"snapshot,omitempty"`
}

type PaymentReport struct {
	ID          string              `json:"id"`
	WindowID    string              `json:"windowId"`
	Source      string              `json:"source"`
	Beneficiary string              `json:"beneficiary"`
	Asset       string              `json:"asset"`
	Amount      amount.Amount       `json:"amount"`
	Settled     amount.Amount       `json:"settled"`
	Outstanding amount.Amount       `json:"outstanding"`
	Fees        amount.Amount       `json:"fees"`
	Status      PaymentStatus       `json:"status"`
	Tranches    []SettlementTranche `json:"tranches"`
}

type EventReport struct {
	Seq       int               `json:"seq"`
	Epoch     int               `json:"epoch"`
	Kind      EventKind         `json:"kind"`
	WindowID  string            `json:"windowId,omitempty"`
	AccountID string            `json:"accountId,omitempty"`
	PaymentID string            `json:"paymentId,omitempty"`
	Asset     string            `json:"asset,omitempty"`
	Amount    amount.Amount     `json:"amount,omitempty"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type ReconciliationLine struct {
	Asset          string        `json:"asset"`
	Reserve        amount.Amount `json:"reserve"`
	FreeClaims     amount.Amount `json:"freeClaims"`
	ReservedClaims amount.Amount `json:"reservedClaims"`
	Outstanding    amount.Amount `json:"outstanding"`
	AccruedFees    amount.Amount `json:"accruedFees"`
	WindowCount    int           `json:"windowCount"`
	PaymentCount   int           `json:"paymentCount"`
	ClosedMargin   int64         `json:"closedMargin"`
}

type PoolReport struct {
	Name           string                   `json:"name"`
	Epoch          int                      `json:"epoch"`
	Assets         []AssetReport            `json:"assets"`
	Accounts       []AccountReport          `json:"accounts"`
	Windows        []WindowReport           `json:"windows"`
	Payments       []PaymentReport          `json:"payments"`
	Reconciliation []ReconciliationLine     `json:"reconciliation"`
	Metrics        map[string]amount.Amount `json:"metrics"`
	Events         []EventReport            `json:"events,omitempty"`
}

func Entries(vector amount.Vector) []AmountEntry {
	keys := make([]string, 0, len(vector))
	for key, value := range vector {
		if value > 0 {
			keys = append(keys, key)
		}
	}
	sortStrings(keys)
	out := make([]AmountEntry, 0, len(keys))
	for _, key := range keys {
		out = append(out, AmountEntry{ID: key, Amount: vector[key]})
	}
	return out
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func AccountToReport(account *Account, assets []string) AccountReport {
	totalClaims := amount.NewVector()
	for _, asset := range assets {
		claim := account.Balances.TotalClaims(asset)
		if claim > 0 {
			totalClaims[asset] = claim
		}
	}
	return AccountReport{
		ID:          account.ID,
		Role:        account.Role,
		Free:        Entries(account.Balances.Free),
		Reserved:    Entries(account.Balances.Reserved),
		SettledIn:   Entries(account.Balances.SettledIn),
		Withdrawn:   Entries(account.Balances.Withdrawn),
		FeesPaid:    Entries(account.Balances.FeesPaid),
		TotalClaims: Entries(totalClaims),
	}
}

func SnapshotToReport(snapshot *WindowSnapshot) *SnapshotReport {
	if snapshot == nil {
		return nil
	}
	return &SnapshotReport{
		Epoch:              snapshot.Epoch,
		Reserve:            snapshot.Reserve,
		Outstanding:        snapshot.Outstanding,
		AccruedFees:        snapshot.AccruedFees,
		SettlementVolume:   snapshot.SettlementVolume,
		SolvencyMargin:     snapshot.SolvencyMargin,
		ClosedPaymentCount: snapshot.ClosedPaymentCount,
		OpenPaymentCount:   snapshot.OpenPaymentCount,
		AccountFree:        Entries(snapshot.AccountFree),
		AccountReserved:    Entries(snapshot.AccountReserved),
	}
}

func WindowToReport(window *Window, payments map[string]*Payment) WindowReport {
	return WindowReport{
		ID:                 window.ID,
		Asset:              window.Asset,
		Status:             window.Status,
		OpenEpoch:          window.OpenEpoch,
		CloseEpoch:         window.CloseEpoch,
		Reserve:            window.Reserve,
		GrossDeposits:      window.GrossDeposits,
		GrossWithdrawals:   window.GrossWithdrawals,
		SettlementVolume:   window.SettlementVolume,
		AccruedFees:        window.AccruedFees,
		Outstanding:        window.Outstanding(payments),
		PaymentCount:       len(window.Payments),
		PostCloseWithdrawn: Entries(window.PostCloseWithdrawn),
		Snapshot:           SnapshotToReport(window.Snapshot),
	}
}

func PaymentToReport(payment *Payment) PaymentReport {
	return PaymentReport{
		ID:          payment.ID,
		WindowID:    payment.WindowID,
		Source:      payment.Source,
		Beneficiary: payment.Beneficiary,
		Asset:       payment.Asset,
		Amount:      payment.Amount,
		Settled:     payment.Settled,
		Outstanding: payment.Outstanding(),
		Fees:        payment.Fees,
		Status:      payment.Status,
		Tranches:    append([]SettlementTranche(nil), payment.Tranches...),
	}
}

func EventToReport(event Event) EventReport {
	return EventReport{
		Seq:       event.Seq,
		Epoch:     event.Epoch,
		Kind:      event.Kind,
		WindowID:  event.WindowID,
		AccountID: event.AccountID,
		PaymentID: event.PaymentID,
		Asset:     event.Asset,
		Amount:    event.Amount,
		Message:   event.Message,
		Fields:    event.Fields,
	}
}
