package engine

import (
	"solsticedtl/src/amount"
	"solsticedtl/src/domain"
	"solsticedtl/src/solvency"
)

func (s *Service) Report(includeEvents bool) domain.PoolReport {
	assetIDs := solvency.AssetIDs(s.state)
	report := domain.PoolReport{
		Name:           s.state.Name,
		Epoch:          s.state.Epoch,
		Assets:         []domain.AssetReport{},
		Accounts:       []domain.AccountReport{},
		Windows:        []domain.WindowReport{},
		Payments:       []domain.PaymentReport{},
		Reconciliation: solvency.Reconciliation(s.state),
		Metrics:        make(map[string]amount.Amount),
	}
	for _, asset := range s.state.SortedAssets() {
		report.Assets = append(report.Assets, domain.AssetReport{
			ID:                 asset.ID,
			Symbol:             asset.Symbol,
			Decimals:           asset.Decimals,
			SettlementPriority: asset.SettlementPriority,
			Active:             asset.Active,
		})
	}
	for _, account := range s.state.SortedAccounts() {
		report.Accounts = append(report.Accounts, domain.AccountToReport(account, assetIDs))
	}
	for _, window := range s.state.SortedWindows() {
		report.Windows = append(report.Windows, domain.WindowToReport(window, s.state.Payments))
	}
	for _, payment := range s.state.SortedPayments() {
		report.Payments = append(report.Payments, domain.PaymentToReport(payment))
	}
	for key, value := range s.state.Metrics {
		report.Metrics[key] = value
	}
	report.Metrics["journalEntries"] = amount.Must(int64(len(s.journal.Entries())))
	if includeEvents {
		report.Events = make([]domain.EventReport, 0, len(s.state.Events))
		for _, event := range s.state.Events {
			report.Events = append(report.Events, domain.EventToReport(event))
		}
	}
	return report
}

func (s *Service) Validate() []string {
	messages := []string{}
	for _, asset := range solvency.AssetIDs(s.state) {
		reserve := s.state.TotalReserve(asset)
		outstanding := s.state.TotalOutstanding(asset)
		reserved := s.state.TotalReservedClaims(asset)
		if reserved < outstanding {
			messages = append(messages, "reserved claims below outstanding obligations for "+asset)
		}
		if reserve == 0 && (outstanding > 0 || reserved > 0) {
			messages = append(messages, "empty reserve carries active claims for "+asset)
		}
	}
	for _, window := range s.state.SortedWindows() {
		if window.IsClosed() && window.Snapshot == nil {
			messages = append(messages, "closed window without snapshot: "+window.ID)
		}
		for _, paymentID := range window.Payments {
			if s.state.Payments[paymentID] == nil {
				messages = append(messages, "window references missing payment: "+paymentID)
			}
		}
	}
	if len(messages) == 0 {
		messages = append(messages, "ok")
	}
	for _, message := range messages {
		s.addValidation(message)
	}
	return messages
}
