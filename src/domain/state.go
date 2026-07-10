package domain

import (
	"fmt"
	"sort"

	"solsticedtl/src/amount"
)

type PoolState struct {
	Name      string
	Epoch     int
	Assets    map[string]*Asset
	Accounts  map[string]*Account
	Windows   map[string]*Window
	Payments  map[string]*Payment
	Events    []Event
	Operators map[string]bool
	Metrics   map[string]amount.Amount
}

func NewPoolState(name string) *PoolState {
	return &PoolState{
		Name:      name,
		Epoch:     1,
		Assets:    make(map[string]*Asset),
		Accounts:  make(map[string]*Account),
		Windows:   make(map[string]*Window),
		Payments:  make(map[string]*Payment),
		Events:    []Event{},
		Operators: make(map[string]bool),
		Metrics:   make(map[string]amount.Amount),
	}
}

func (s *PoolState) Clone() *PoolState {
	out := NewPoolState(s.Name)
	out.Epoch = s.Epoch
	for id, asset := range s.Assets {
		out.Assets[id] = asset.Clone()
	}
	for id, account := range s.Accounts {
		out.Accounts[id] = account.Clone()
	}
	for id, window := range s.Windows {
		out.Windows[id] = window.Clone()
	}
	for id, payment := range s.Payments {
		out.Payments[id] = payment.Clone()
	}
	out.Events = make([]Event, 0, len(s.Events))
	for _, event := range s.Events {
		out.Events = append(out.Events, event.Clone())
	}
	for id, active := range s.Operators {
		out.Operators[id] = active
	}
	for key, value := range s.Metrics {
		out.Metrics[key] = value
	}
	return out
}

func (s *PoolState) AddMetric(key string, value amount.Amount) {
	s.Metrics[key] = s.Metrics[key].MustAdd(value)
}

func (s *PoolState) SetMetric(key string, value amount.Amount) {
	s.Metrics[key] = value
}

func (s *PoolState) NextEvent(kind EventKind) Event {
	return NewEvent(len(s.Events)+1, s.Epoch, kind)
}

func (s *PoolState) AppendEvent(event Event) {
	if event.Seq == 0 {
		event.Seq = len(s.Events) + 1
	}
	if event.Epoch == 0 {
		event.Epoch = s.Epoch
	}
	s.Events = append(s.Events, event)
}

func (s *PoolState) RequireAsset(id string) (*Asset, error) {
	asset := s.Assets[id]
	if asset == nil {
		return nil, fmt.Errorf("unknown asset %s", id)
	}
	if !asset.Active {
		return nil, fmt.Errorf("asset %s is inactive", id)
	}
	return asset, nil
}

func (s *PoolState) RequireAccount(id string) (*Account, error) {
	account := s.Accounts[id]
	if account == nil {
		return nil, fmt.Errorf("unknown account %s", id)
	}
	if account.Disabled {
		return nil, fmt.Errorf("account %s is disabled", id)
	}
	return account, nil
}

func (s *PoolState) RequireWindow(id string) (*Window, error) {
	window := s.Windows[id]
	if window == nil {
		return nil, fmt.Errorf("unknown window %s", id)
	}
	return window, nil
}

func (s *PoolState) RequirePayment(id string) (*Payment, error) {
	payment := s.Payments[id]
	if payment == nil {
		return nil, fmt.Errorf("unknown payment %s", id)
	}
	return payment, nil
}

func (s *PoolState) AddAsset(asset *Asset) error {
	if asset == nil {
		return fmt.Errorf("asset is nil")
	}
	if _, exists := s.Assets[asset.ID]; exists {
		return fmt.Errorf("asset %s already exists", asset.ID)
	}
	s.Assets[asset.ID] = asset
	event := s.NextEvent(EventAssetListed)
	event.Asset = asset.ID
	event.Message = "asset listed for solar window settlement"
	s.AppendEvent(event)
	return nil
}

func (s *PoolState) AddAccount(account *Account) error {
	if account == nil {
		return fmt.Errorf("account is nil")
	}
	if _, exists := s.Accounts[account.ID]; exists {
		return fmt.Errorf("account %s already exists", account.ID)
	}
	s.Accounts[account.ID] = account
	if account.Role == RoleOperator {
		s.Operators[account.ID] = true
	}
	event := s.NextEvent(EventAccountCreated)
	event.AccountID = account.ID
	event.Message = "account registered"
	s.AppendEvent(event)
	return nil
}

func (s *PoolState) AddWindow(window *Window) error {
	if window == nil {
		return fmt.Errorf("window is nil")
	}
	if _, exists := s.Windows[window.ID]; exists {
		return fmt.Errorf("window %s already exists", window.ID)
	}
	if _, err := s.RequireAsset(window.Asset); err != nil {
		return err
	}
	s.Windows[window.ID] = window
	event := s.NextEvent(EventWindowOpened)
	event.WindowID = window.ID
	event.Asset = window.Asset
	event.Message = "solar liquidity window opened"
	s.AppendEvent(event)
	return nil
}

func (s *PoolState) AddPayment(payment *Payment) error {
	if payment == nil {
		return fmt.Errorf("payment is nil")
	}
	if _, exists := s.Payments[payment.ID]; exists {
		return fmt.Errorf("payment %s already exists", payment.ID)
	}
	window, err := s.RequireWindow(payment.WindowID)
	if err != nil {
		return err
	}
	if window.Asset != payment.Asset {
		return fmt.Errorf("payment asset does not match window asset")
	}
	s.Payments[payment.ID] = payment
	window.AddPayment(payment.ID)
	event := s.NextEvent(EventPaymentQueued)
	event.PaymentID = payment.ID
	event.WindowID = payment.WindowID
	event.AccountID = payment.Source
	event.Asset = payment.Asset
	event.Amount = payment.Amount
	event.Message = "deferred payment queued"
	s.AppendEvent(event)
	return nil
}

func (s *PoolState) SortedAssets() []*Asset {
	keys := make([]string, 0, len(s.Assets))
	for key := range s.Assets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*Asset, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.Assets[key])
	}
	return out
}

func (s *PoolState) SortedAccounts() []*Account {
	keys := make([]string, 0, len(s.Accounts))
	for key := range s.Accounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*Account, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.Accounts[key])
	}
	return out
}

func (s *PoolState) SortedWindows() []*Window {
	keys := make([]string, 0, len(s.Windows))
	for key := range s.Windows {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*Window, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.Windows[key])
	}
	return out
}

func (s *PoolState) SortedPayments() []*Payment {
	keys := make([]string, 0, len(s.Payments))
	for key := range s.Payments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*Payment, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.Payments[key])
	}
	return out
}

func (s *PoolState) PaymentsForWindow(windowID string) []*Payment {
	window := s.Windows[windowID]
	if window == nil {
		return nil
	}
	ids := append([]string(nil), window.Payments...)
	sort.Strings(ids)
	out := make([]*Payment, 0, len(ids))
	for _, id := range ids {
		if payment := s.Payments[id]; payment != nil {
			out = append(out, payment)
		}
	}
	return out
}

func (s *PoolState) TotalReserve(asset string) amount.Amount {
	var total amount.Amount
	for _, window := range s.Windows {
		if window.Asset != asset {
			continue
		}
		total = total.MustAdd(window.Reserve)
	}
	return total
}

func (s *PoolState) TotalOutstanding(asset string) amount.Amount {
	var total amount.Amount
	for _, payment := range s.Payments {
		if payment.Asset != asset || !payment.IsOpen() {
			continue
		}
		total = total.MustAdd(payment.Outstanding())
	}
	return total
}

func (s *PoolState) TotalFreeClaims(asset string) amount.Amount {
	var total amount.Amount
	for _, account := range s.Accounts {
		total = total.MustAdd(account.Balances.Free.Get(asset))
	}
	return total
}

func (s *PoolState) TotalReservedClaims(asset string) amount.Amount {
	var total amount.Amount
	for _, account := range s.Accounts {
		total = total.MustAdd(account.Balances.Reserved.Get(asset))
	}
	return total
}

func (s *PoolState) TreasuryAccount() *Account {
	for _, account := range s.Accounts {
		if account.Role == RoleTreasury {
			return account
		}
	}
	return nil
}

func (s *PoolState) ActiveOperator(id string) bool {
	return s.Operators[id]
}
