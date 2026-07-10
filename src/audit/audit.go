package audit

import (
	"fmt"
	"sort"
	"strings"

	"solsticedtl/src/amount"
	"solsticedtl/src/domain"
	"solsticedtl/src/solvency"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityNotice  Severity = "notice"
	SeverityWarning Severity = "warning"
)

type Finding struct {
	Severity  Severity `json:"severity"`
	Subject   string   `json:"subject"`
	Message   string   `json:"message"`
	Reference string   `json:"reference,omitempty"`
}

type WindowScore struct {
	WindowID            string        `json:"windowId"`
	Asset               string        `json:"asset"`
	Status              string        `json:"status"`
	Reserve             amount.Amount `json:"reserve"`
	Outstanding         amount.Amount `json:"outstanding"`
	FreeClaims          amount.Amount `json:"freeClaims"`
	ReservedClaims      amount.Amount `json:"reservedClaims"`
	FeeLoad             amount.Amount `json:"feeLoad"`
	UtilizationBps      int64         `json:"utilizationBps"`
	SettlementBps       int64         `json:"settlementBps"`
	PostCloseReleaseBps int64         `json:"postCloseReleaseBps"`
}

type AccountScore struct {
	AccountID    string        `json:"accountId"`
	Asset        string        `json:"asset"`
	Free         amount.Amount `json:"free"`
	Reserved     amount.Amount `json:"reserved"`
	SettledIn    amount.Amount `json:"settledIn"`
	Withdrawn    amount.Amount `json:"withdrawn"`
	ReserveShare int64         `json:"reserveShare"`
}

type Report struct {
	Name      string                   `json:"name"`
	Epoch     int                      `json:"epoch"`
	Windows   []WindowScore            `json:"windows"`
	Accounts  []AccountScore           `json:"accounts"`
	Findings  []Finding                `json:"findings"`
	Summaries map[string]amount.Amount `json:"summaries"`
}

func Build(state *domain.PoolState) Report {
	report := Report{
		Name:      state.Name,
		Epoch:     state.Epoch,
		Windows:   []WindowScore{},
		Accounts:  []AccountScore{},
		Findings:  []Finding{},
		Summaries: make(map[string]amount.Amount),
	}
	for _, window := range state.SortedWindows() {
		score := scoreWindow(state, window)
		report.Windows = append(report.Windows, score)
		report.Summaries["reserve"] = report.Summaries["reserve"].MustAdd(score.Reserve)
		report.Summaries["outstanding"] = report.Summaries["outstanding"].MustAdd(score.Outstanding)
		report.Summaries["fees"] = report.Summaries["fees"].MustAdd(score.FeeLoad)
		report.Findings = append(report.Findings, inspectWindow(state, window, score)...)
	}
	for _, account := range state.SortedAccounts() {
		report.Accounts = append(report.Accounts, scoreAccount(state, account)...)
	}
	report.Findings = append(report.Findings, inspectAssets(state)...)
	sort.Slice(report.Findings, func(i, j int) bool {
		left := string(report.Findings[i].Severity) + ":" + report.Findings[i].Subject
		right := string(report.Findings[j].Severity) + ":" + report.Findings[j].Subject
		return left < right
	})
	return report
}

func scoreWindow(state *domain.PoolState, window *domain.Window) WindowScore {
	view := solvency.View(state, window)
	score := WindowScore{
		WindowID:       window.ID,
		Asset:          window.Asset,
		Status:         string(window.Status),
		Reserve:        view.Reserve,
		Outstanding:    view.Outstanding,
		FreeClaims:     view.FreeClaims,
		ReservedClaims: view.ReservedClaims,
		FeeLoad:        view.AccruedFees,
	}
	if window.GrossDeposits > 0 {
		score.UtilizationBps = bps(view.Outstanding.MustAdd(window.SettlementVolume), window.GrossDeposits)
		score.SettlementBps = bps(window.SettlementVolume, window.GrossDeposits)
		score.PostCloseReleaseBps = bps(view.PostCloseWithdrawn, window.GrossDeposits)
	}
	return score
}

func scoreAccount(state *domain.PoolState, account *domain.Account) []AccountScore {
	assetIDs := solvency.AssetIDs(state)
	out := make([]AccountScore, 0, len(assetIDs))
	for _, asset := range assetIDs {
		free := account.Balances.Free.Get(asset)
		reserved := account.Balances.Reserved.Get(asset)
		settled := account.Balances.SettledIn.Get(asset)
		withdrawn := account.Balances.Withdrawn.Get(asset)
		if free == 0 && reserved == 0 && settled == 0 && withdrawn == 0 {
			continue
		}
		totalReserve := state.TotalReserve(asset)
		score := AccountScore{
			AccountID: account.ID,
			Asset:     asset,
			Free:      free,
			Reserved:  reserved,
			SettledIn: settled,
			Withdrawn: withdrawn,
		}
		if totalReserve > 0 {
			score.ReserveShare = bps(free.MustAdd(reserved), totalReserve)
		}
		out = append(out, score)
	}
	return out
}

func inspectWindow(state *domain.PoolState, window *domain.Window, score WindowScore) []Finding {
	findings := []Finding{}
	if window.IsOpen() && state.Epoch > window.CloseEpoch {
		findings = append(findings, Finding{
			Severity:  SeverityNotice,
			Subject:   window.ID,
			Message:   "window remains open after scheduled close epoch",
			Reference: "window.closeEpoch",
		})
	}
	if window.IsClosed() && window.Snapshot == nil {
		findings = append(findings, Finding{
			Severity:  SeverityWarning,
			Subject:   window.ID,
			Message:   "closed window has no recorded close snapshot",
			Reference: "window.snapshot",
		})
	}
	if score.UtilizationBps > 9_500 {
		findings = append(findings, Finding{
			Severity:  SeverityNotice,
			Subject:   window.ID,
			Message:   "window utilization is above operating target",
			Reference: "window.utilization",
		})
	}
	if score.FeeLoad > 0 && score.SettlementBps == 0 {
		findings = append(findings, Finding{
			Severity:  SeverityInfo,
			Subject:   window.ID,
			Message:   "fee load recorded before settlement ratio changed",
			Reference: "window.fees",
		})
	}
	for _, paymentID := range window.Payments {
		payment := state.Payments[paymentID]
		if payment == nil {
			findings = append(findings, Finding{
				Severity:  SeverityWarning,
				Subject:   window.ID,
				Message:   "window references missing payment",
				Reference: paymentID,
			})
			continue
		}
		if payment.Status == domain.PaymentQueued && state.Epoch > payment.DueEpoch+3 {
			findings = append(findings, Finding{
				Severity:  SeverityNotice,
				Subject:   payment.ID,
				Message:   "queued payment is past target settlement epoch",
				Reference: "payment.dueEpoch",
			})
		}
	}
	return findings
}

func inspectAssets(state *domain.PoolState) []Finding {
	findings := []Finding{}
	for _, asset := range state.SortedAssets() {
		if !asset.Active {
			findings = append(findings, Finding{
				Severity: SeverityInfo,
				Subject:  asset.ID,
				Message:  "asset is inactive",
			})
		}
		if asset.Decimals == 0 {
			findings = append(findings, Finding{
				Severity:  SeverityNotice,
				Subject:   asset.ID,
				Message:   "asset uses integer settlement precision",
				Reference: "asset.decimals",
			})
		}
	}
	return findings
}

func bps(numerator amount.Amount, denominator amount.Amount) int64 {
	if denominator == 0 || numerator == 0 {
		return 0
	}
	return int64(numerator) * amount.BasisPoints / int64(denominator)
}

func Format(report Report) string {
	lines := []string{
		fmt.Sprintf("audit report: %s epoch=%d", report.Name, report.Epoch),
	}
	for _, window := range report.Windows {
		lines = append(lines, fmt.Sprintf(
			"window=%s asset=%s status=%s reserve=%s outstanding=%s utilization=%dbps",
			window.WindowID,
			window.Asset,
			window.Status,
			window.Reserve.String(),
			window.Outstanding.String(),
			window.UtilizationBps,
		))
	}
	if len(report.Findings) == 0 {
		lines = append(lines, "findings=none")
	} else {
		for _, finding := range report.Findings {
			lines = append(lines, fmt.Sprintf(
				"finding severity=%s subject=%s message=%s",
				finding.Severity,
				finding.Subject,
				finding.Message,
			))
		}
	}
	return strings.Join(lines, "\n")
}
