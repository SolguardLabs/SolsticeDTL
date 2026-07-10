package engine

import (
	"fmt"
	"strings"

	"solsticedtl/src/amount"
	"solsticedtl/src/domain"
	"solsticedtl/src/ids"
	"solsticedtl/src/ledger"
	"solsticedtl/src/policy"
	"solsticedtl/src/solvency"
)

type Service struct {
	state       *domain.PoolState
	policy      policy.Policy
	journal     *ledger.Journal
	validations []string
}

func New(name string, rules policy.Policy) (*Service, error) {
	if strings.TrimSpace(name) == "" {
		name = "SolsticeDTL"
	}
	if err := rules.Validate(); err != nil {
		return nil, err
	}
	return &Service{
		state:       domain.NewPoolState(name),
		policy:      rules,
		journal:     ledger.NewJournal(),
		validations: []string{},
	}, nil
}

func MustNew(name string, rules policy.Policy) *Service {
	service, err := New(name, rules)
	if err != nil {
		panic(err)
	}
	return service
}

func (s *Service) State() *domain.PoolState {
	return s.state
}

func (s *Service) Policy() policy.Policy {
	return s.policy
}

func (s *Service) Journal() *ledger.Journal {
	return s.journal
}

func (s *Service) Validations() []string {
	return append([]string(nil), s.validations...)
}

func (s *Service) addValidation(message string) {
	s.validations = append(s.validations, message)
}

func (s *Service) RegisterAsset(req AssetRequest) error {
	id, err := ids.NewAssetID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	symbol := strings.TrimSpace(req.Symbol)
	if symbol == "" {
		symbol = strings.ToUpper(id.String())
	}
	if req.Decimals < 0 || req.Decimals > 18 {
		return invalid("asset decimals out of range")
	}
	asset := domain.NewAsset(id.String(), symbol, req.Decimals)
	if req.SettlementPriority > 0 {
		asset.SettlementPriority = req.SettlementPriority
	}
	for key, value := range req.Metadata {
		asset.Metadata[key] = value
	}
	return s.state.AddAsset(asset)
}

func roleFromString(raw string) domain.AccountRole {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "merchant":
		return domain.RoleMerchant
	case "operator":
		return domain.RoleOperator
	case "treasury":
		return domain.RoleTreasury
	default:
		return domain.RoleUser
	}
}

func (s *Service) RegisterAccount(req AccountRequest) error {
	id, err := ids.NewAccountID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	account := domain.NewAccount(id.String(), roleFromString(req.Role))
	account.DisplayName = strings.TrimSpace(req.DisplayName)
	for key, value := range req.Metadata {
		account.Metadata[key] = value
	}
	return s.state.AddAccount(account)
}

func (s *Service) OpenWindow(req WindowRequest) error {
	id, err := ids.NewWindowID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	assetID, err := ids.NewAssetID(req.Asset)
	if err != nil {
		return invalid(err.Error())
	}
	if req.OpenEpoch <= 0 {
		req.OpenEpoch = s.state.Epoch
	}
	if req.CloseEpoch <= req.OpenEpoch {
		req.CloseEpoch = req.OpenEpoch + 1
	}
	window := domain.NewWindow(id.String(), assetID.String(), req.OpenEpoch, req.CloseEpoch)
	window.Label = strings.TrimSpace(req.Label)
	for key, value := range req.Metadata {
		window.Metadata[key] = value
	}
	return s.state.AddWindow(window)
}

func (s *Service) Advance(req AdvanceRequest) error {
	if req.To > 0 {
		if req.To < s.state.Epoch {
			return stateError("cannot move epoch backwards")
		}
		s.state.Epoch = req.To
		return nil
	}
	if req.Epochs <= 0 {
		req.Epochs = 1
	}
	s.state.Epoch += req.Epochs
	return nil
}

func (s *Service) requireWindowAsset(window *domain.Window, asset string) error {
	if window.Asset != asset {
		return invalid("asset %s does not match window %s asset %s", asset, window.ID, window.Asset)
	}
	_, err := s.state.RequireAsset(asset)
	if err != nil {
		return notFound(err.Error())
	}
	return nil
}

func (s *Service) Deposit(req DepositRequest) error {
	accountID, err := ids.NewAccountID(req.AccountID)
	if err != nil {
		return invalid(err.Error())
	}
	windowID, err := ids.NewWindowID(req.WindowID)
	if err != nil {
		return invalid(err.Error())
	}
	assetID, err := ids.NewAssetID(req.Asset)
	if err != nil {
		return invalid(err.Error())
	}
	if err := s.policy.CheckDeposit(req.Amount); err != nil {
		return policyError(err.Error())
	}
	account, err := s.state.RequireAccount(accountID.String())
	if err != nil {
		return notFound(err.Error())
	}
	window, err := s.state.RequireWindow(windowID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if !window.IsOpen() {
		return stateError("window %s is not open", window.ID)
	}
	if err := s.requireWindowAsset(window, assetID.String()); err != nil {
		return err
	}
	if err := s.policy.CheckWindowReserve(window.Reserve, req.Amount); err != nil {
		return policyError(err.Error())
	}
	entries, err := ledger.Deposit(s.state.Epoch, account, window, assetID.String(), req.Amount)
	if err != nil {
		return stateError(err.Error())
	}
	ledger.AppendAll(s.journal, entries)
	s.state.AddMetric("deposits", req.Amount)
	event := s.state.NextEvent(domain.EventDeposit)
	event.AccountID = account.ID
	event.WindowID = window.ID
	event.Asset = assetID.String()
	event.Amount = req.Amount
	event.Message = "reserve deposit accepted"
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) QueuePayment(req QueuePaymentRequest) error {
	paymentID, err := ids.NewPaymentID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	windowID, err := ids.NewWindowID(req.WindowID)
	if err != nil {
		return invalid(err.Error())
	}
	sourceID, err := ids.NewAccountID(req.Source)
	if err != nil {
		return invalid(err.Error())
	}
	beneficiaryID, err := ids.NewAccountID(req.Beneficiary)
	if err != nil {
		return invalid(err.Error())
	}
	assetID, err := ids.NewAssetID(req.Asset)
	if err != nil {
		return invalid(err.Error())
	}
	if err := s.policy.CheckPayment(req.Amount); err != nil {
		return policyError(err.Error())
	}
	window, err := s.state.RequireWindow(windowID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if !window.IsOpen() {
		return stateError("window %s is not open", window.ID)
	}
	if err := s.requireWindowAsset(window, assetID.String()); err != nil {
		return err
	}
	source, err := s.state.RequireAccount(sourceID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if _, err := s.state.RequireAccount(beneficiaryID.String()); err != nil {
		return notFound(err.Error())
	}
	if source.Available(assetID.String()) < req.Amount {
		return solvencyError("source free balance cannot cover payment")
	}
	payment := domain.NewPayment(paymentID.String(), window.ID, source.ID, beneficiaryID.String(), assetID.String(), req.Amount, s.state.Epoch)
	if req.DueEpoch > 0 {
		payment.DueEpoch = req.DueEpoch
	}
	if req.Priority > 0 {
		payment.Priority = req.Priority
	}
	for key, value := range req.Metadata {
		payment.Metadata[key] = value
	}
	entries, err := ledger.QueuePayment(s.state.Epoch, source, payment)
	if err != nil {
		return stateError(err.Error())
	}
	if err := s.state.AddPayment(payment); err != nil {
		return stateError(err.Error())
	}
	ledger.AppendAll(s.journal, entries)
	s.state.AddMetric("queuedPayments", req.Amount)
	return nil
}

func (s *Service) CloseWindow(req CloseWindowRequest) error {
	windowID, err := ids.NewWindowID(req.WindowID)
	if err != nil {
		return invalid(err.Error())
	}
	window, err := s.state.RequireWindow(windowID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if !window.IsOpen() {
		return stateError("window %s is not open", window.ID)
	}
	if s.state.Epoch < window.CloseEpoch {
		return stateError("window %s cannot close before epoch %d", window.ID, window.CloseEpoch)
	}
	ok, snapshot := solvency.CanClose(s.state, window)
	if !ok && s.policy.CloseRequiresSolvency {
		return solvencyError("window %s is below closing margin", window.ID)
	}
	window.Snapshot = snapshot
	window.Status = domain.WindowClosed
	s.state.SetMetric("lastClosedEpoch", amount.Must(int64(s.state.Epoch)))
	event := s.state.NextEvent(domain.EventWindowClosed)
	event.WindowID = window.ID
	event.Asset = window.Asset
	event.Amount = snapshot.Reserve
	event.Message = "window closed with reserve snapshot"
	event.Fields["openPayments"] = fmt.Sprintf("%d", snapshot.OpenPaymentCount)
	event.Fields["closedMargin"] = fmt.Sprintf("%d", snapshot.SolvencyMargin)
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) Settle(req SettlementRequest) error {
	paymentID, err := ids.NewPaymentID(req.PaymentID)
	if err != nil {
		return invalid(err.Error())
	}
	payment, err := s.state.RequirePayment(paymentID.String())
	if err != nil {
		return notFound(err.Error())
	}
	windowID := payment.WindowID
	if strings.TrimSpace(req.WindowID) != "" {
		parsedWindowID, err := ids.NewWindowID(req.WindowID)
		if err != nil {
			return invalid(err.Error())
		}
		windowID = parsedWindowID.String()
	}
	if windowID != payment.WindowID {
		return invalid("payment %s is not assigned to window %s", payment.ID, windowID)
	}
	window, err := s.state.RequireWindow(windowID)
	if err != nil {
		return notFound(err.Error())
	}
	if !window.IsClosed() {
		return stateError("window %s is not closed", window.ID)
	}
	if !payment.IsOpen() {
		return stateError("payment %s is not open", payment.ID)
	}
	if req.Amount == 0 {
		req.Amount = payment.Outstanding()
	}
	maxPartial, err := s.policy.MaxPartialFor(payment.Outstanding())
	if err != nil {
		return policyError(err.Error())
	}
	if req.Amount > maxPartial {
		return policyError("settlement exceeds partial limit")
	}
	if req.Amount > payment.Outstanding() {
		return stateError("settlement exceeds payment outstanding")
	}
	fee, err := s.policy.SettlementFee(req.Amount)
	if err != nil {
		return policyError(err.Error())
	}
	if req.Amount.MustAdd(fee) > window.Reserve {
		return solvencyError("window reserve cannot cover settlement")
	}
	source, err := s.state.RequireAccount(payment.Source)
	if err != nil {
		return notFound(err.Error())
	}
	beneficiary, err := s.state.RequireAccount(payment.Beneficiary)
	if err != nil {
		return notFound(err.Error())
	}
	operator := strings.TrimSpace(req.Operator)
	if operator == "" {
		operator = "scheduler"
	}
	entries, tranche, err := ledger.SettlePayment(s.state.Epoch, source, beneficiary, window, payment, req.Amount, fee, operator)
	if err != nil {
		return stateError(err.Error())
	}
	ledger.AppendAll(s.journal, entries)
	s.state.AddMetric("settledGross", req.Amount)
	s.state.AddMetric("settlementFees", fee)
	event := s.state.NextEvent(domain.EventSettlement)
	event.WindowID = window.ID
	event.PaymentID = payment.ID
	event.AccountID = payment.Beneficiary
	event.Asset = payment.Asset
	event.Amount = tranche.Net
	event.Message = "payment tranche settled"
	event.Fields["gross"] = req.Amount.String()
	event.Fields["fee"] = fee.String()
	event.Fields["operator"] = operator
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) Withdraw(req WithdrawalRequest) error {
	accountID, err := ids.NewAccountID(req.AccountID)
	if err != nil {
		return invalid(err.Error())
	}
	windowID, err := ids.NewWindowID(req.WindowID)
	if err != nil {
		return invalid(err.Error())
	}
	assetID, err := ids.NewAssetID(req.Asset)
	if err != nil {
		return invalid(err.Error())
	}
	if err := s.policy.CheckWithdrawal(req.Amount); err != nil {
		return policyError(err.Error())
	}
	account, err := s.state.RequireAccount(accountID.String())
	if err != nil {
		return notFound(err.Error())
	}
	window, err := s.state.RequireWindow(windowID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if err := s.requireWindowAsset(window, assetID.String()); err != nil {
		return err
	}
	if window.IsClosed() && !s.policy.AllowPostCloseWithdrawal {
		return policyError("post-close withdrawals disabled")
	}
	if account.Available(assetID.String()) < req.Amount {
		return solvencyError("account free balance cannot cover withdrawal")
	}
	if window.Reserve < req.Amount {
		return solvencyError("window reserve cannot cover withdrawal")
	}
	if window.IsClosed() {
		capacity := solvency.RemainingSnapshotCapacity(window, account.ID)
		if req.Amount > capacity {
			return solvencyError("withdrawal exceeds window capacity")
		}
	} else if req.Amount > account.Available(assetID.String()) {
		return solvencyError("withdrawal exceeds live account capacity")
	}
	entries, err := ledger.Withdraw(s.state.Epoch, account, window, assetID.String(), req.Amount)
	if err != nil {
		return stateError(err.Error())
	}
	ledger.AppendAll(s.journal, entries)
	s.state.AddMetric("withdrawals", req.Amount)
	event := s.state.NextEvent(domain.EventWithdrawal)
	event.WindowID = window.ID
	event.AccountID = account.ID
	event.Asset = assetID.String()
	event.Amount = req.Amount
	event.Message = "account withdrawal processed"
	s.state.AppendEvent(event)
	return nil
}
