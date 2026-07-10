package domain

import (
	"fmt"
	"sort"
	"time"

	"solsticedtl/src/amount"
)

type AccountRole string

const (
	RoleUser     AccountRole = "user"
	RoleMerchant AccountRole = "merchant"
	RoleOperator AccountRole = "operator"
	RoleTreasury AccountRole = "treasury"
)

type WindowStatus string

const (
	WindowDraft    WindowStatus = "draft"
	WindowOpen     WindowStatus = "open"
	WindowClosed   WindowStatus = "closed"
	WindowArchived WindowStatus = "archived"
)

type PaymentStatus string

const (
	PaymentQueued   PaymentStatus = "queued"
	PaymentSettling PaymentStatus = "settling"
	PaymentSettled  PaymentStatus = "settled"
	PaymentCanceled PaymentStatus = "canceled"
)

type EventKind string

const (
	EventAccountCreated EventKind = "account_created"
	EventAssetListed    EventKind = "asset_listed"
	EventWindowOpened   EventKind = "window_opened"
	EventDeposit        EventKind = "deposit"
	EventPaymentQueued  EventKind = "payment_queued"
	EventWindowClosed   EventKind = "window_closed"
	EventFeeAdjustment  EventKind = "fee_adjustment"
	EventSettlement     EventKind = "settlement"
	EventWithdrawal     EventKind = "withdrawal"
	EventValidation     EventKind = "validation"
)

type Asset struct {
	ID                 string
	Symbol             string
	Decimals           int
	MinUnit            amount.Amount
	SettlementPriority int
	Active             bool
	Metadata           map[string]string
}

func NewAsset(id string, symbol string, decimals int) *Asset {
	return &Asset{
		ID:                 id,
		Symbol:             symbol,
		Decimals:           decimals,
		MinUnit:            amount.One,
		SettlementPriority: 100,
		Active:             true,
		Metadata:           make(map[string]string),
	}
}

func (a *Asset) Clone() *Asset {
	out := *a
	out.Metadata = make(map[string]string)
	for key, value := range a.Metadata {
		out.Metadata[key] = value
	}
	return &out
}

type BalanceSheet struct {
	Free      amount.Vector
	Reserved  amount.Vector
	SettledIn amount.Vector
	Withdrawn amount.Vector
	FeesPaid  amount.Vector
}

func NewBalanceSheet() BalanceSheet {
	return BalanceSheet{
		Free:      amount.NewVector(),
		Reserved:  amount.NewVector(),
		SettledIn: amount.NewVector(),
		Withdrawn: amount.NewVector(),
		FeesPaid:  amount.NewVector(),
	}
}

func (b BalanceSheet) Clone() BalanceSheet {
	return BalanceSheet{
		Free:      b.Free.Clone(),
		Reserved:  b.Reserved.Clone(),
		SettledIn: b.SettledIn.Clone(),
		Withdrawn: b.Withdrawn.Clone(),
		FeesPaid:  b.FeesPaid.Clone(),
	}
}

func (b BalanceSheet) TotalClaims(asset string) amount.Amount {
	total := b.Free.Get(asset)
	total = total.MustAdd(b.Reserved.Get(asset))
	return total
}

func (b BalanceSheet) CreditFree(asset string, value amount.Amount) error {
	return b.Free.Add(asset, value)
}

func (b BalanceSheet) DebitFree(asset string, value amount.Amount) error {
	return b.Free.Sub(asset, value)
}

func (b BalanceSheet) CreditReserved(asset string, value amount.Amount) error {
	return b.Reserved.Add(asset, value)
}

func (b BalanceSheet) DebitReserved(asset string, value amount.Amount) error {
	return b.Reserved.Sub(asset, value)
}

func (b BalanceSheet) RecordWithdraw(asset string, value amount.Amount) error {
	return b.Withdrawn.Add(asset, value)
}

func (b BalanceSheet) RecordFee(asset string, value amount.Amount) error {
	return b.FeesPaid.Add(asset, value)
}

func (b BalanceSheet) RecordSettlementIn(asset string, value amount.Amount) error {
	return b.SettledIn.Add(asset, value)
}

type Account struct {
	ID          string
	DisplayName string
	Role        AccountRole
	Balances    BalanceSheet
	Metadata    map[string]string
	CreatedAt   time.Time
	Disabled    bool
}

func NewAccount(id string, role AccountRole) *Account {
	return &Account{
		ID:        id,
		Role:      role,
		Balances:  NewBalanceSheet(),
		Metadata:  make(map[string]string),
		CreatedAt: time.Unix(0, 0).UTC(),
	}
}

func (a *Account) Clone() *Account {
	out := *a
	out.Balances = a.Balances.Clone()
	out.Metadata = make(map[string]string)
	for key, value := range a.Metadata {
		out.Metadata[key] = value
	}
	return &out
}

func (a *Account) Available(asset string) amount.Amount {
	return a.Balances.Free.Get(asset)
}

func (a *Account) Reserved(asset string) amount.Amount {
	return a.Balances.Reserved.Get(asset)
}

func (a *Account) Deposit(asset string, value amount.Amount) error {
	return a.Balances.CreditFree(asset, value)
}

func (a *Account) Reserve(asset string, value amount.Amount) error {
	if err := a.Balances.DebitFree(asset, value); err != nil {
		return err
	}
	return a.Balances.CreditReserved(asset, value)
}

func (a *Account) ReleaseReserved(asset string, value amount.Amount) error {
	if err := a.Balances.DebitReserved(asset, value); err != nil {
		return err
	}
	return a.Balances.CreditFree(asset, value)
}

func (a *Account) ConsumeReserved(asset string, value amount.Amount) error {
	return a.Balances.DebitReserved(asset, value)
}

func (a *Account) Withdraw(asset string, value amount.Amount) error {
	if err := a.Balances.DebitFree(asset, value); err != nil {
		return err
	}
	return a.Balances.RecordWithdraw(asset, value)
}

func (a *Account) ReceiveSettlement(asset string, value amount.Amount) error {
	return a.Balances.RecordSettlementIn(asset, value)
}

func (a *Account) PayFee(asset string, value amount.Amount) error {
	return a.Balances.RecordFee(asset, value)
}

type SettlementTranche struct {
	Epoch      int
	Amount     amount.Amount
	Fee        amount.Amount
	Net        amount.Amount
	Operator   string
	Reference  string
	RecordedAt int
}

type Payment struct {
	ID          string
	WindowID    string
	Source      string
	Beneficiary string
	Asset       string
	Amount      amount.Amount
	Settled     amount.Amount
	Fees        amount.Amount
	CreatedAt   int
	DueEpoch    int
	Priority    int
	Status      PaymentStatus
	Tranches    []SettlementTranche
	Metadata    map[string]string
}

func NewPayment(id, windowID, source, beneficiary, asset string, value amount.Amount, epoch int) *Payment {
	return &Payment{
		ID:          id,
		WindowID:    windowID,
		Source:      source,
		Beneficiary: beneficiary,
		Asset:       asset,
		Amount:      value,
		CreatedAt:   epoch,
		DueEpoch:    epoch,
		Priority:    100,
		Status:      PaymentQueued,
		Tranches:    []SettlementTranche{},
		Metadata:    make(map[string]string),
	}
}

func (p *Payment) Clone() *Payment {
	out := *p
	out.Tranches = append([]SettlementTranche(nil), p.Tranches...)
	out.Metadata = make(map[string]string)
	for key, value := range p.Metadata {
		out.Metadata[key] = value
	}
	return &out
}

func (p *Payment) Outstanding() amount.Amount {
	if p.Settled >= p.Amount {
		return amount.Zero
	}
	return p.Amount.MustSub(p.Settled)
}

func (p *Payment) IsOpen() bool {
	return p.Status == PaymentQueued || p.Status == PaymentSettling
}

func (p *Payment) ApplySettlement(epoch int, value amount.Amount, fee amount.Amount, operator string) (SettlementTranche, error) {
	if value.IsZero() {
		return SettlementTranche{}, fmt.Errorf("settlement amount is zero")
	}
	if value > p.Outstanding() {
		return SettlementTranche{}, fmt.Errorf("settlement exceeds outstanding amount")
	}
	net, err := value.Sub(fee)
	if err != nil {
		return SettlementTranche{}, err
	}
	p.Settled = p.Settled.MustAdd(value)
	p.Fees = p.Fees.MustAdd(fee)
	if p.Settled == p.Amount {
		p.Status = PaymentSettled
	} else {
		p.Status = PaymentSettling
	}
	tranche := SettlementTranche{
		Epoch:      epoch,
		Amount:     value,
		Fee:        fee,
		Net:        net,
		Operator:   operator,
		Reference:  fmt.Sprintf("%s:%d", p.ID, len(p.Tranches)+1),
		RecordedAt: len(p.Tranches) + 1,
	}
	p.Tranches = append(p.Tranches, tranche)
	return tranche, nil
}

type FeeAdjustment struct {
	Epoch     int
	Asset     string
	Basis     amount.Amount
	Fee       amount.Amount
	Reason    string
	PaymentID string
}

type WindowSnapshot struct {
	WindowID           string
	Epoch              int
	Reserve            amount.Amount
	Outstanding        amount.Amount
	AccruedFees        amount.Amount
	SettlementVolume   amount.Amount
	AccountFree        amount.Vector
	AccountReserved    amount.Vector
	PaymentOutstanding amount.Vector
	SolvencyMargin     int64
	ClosedPaymentCount int
	OpenPaymentCount   int
}

func NewWindowSnapshot(windowID string, epoch int) *WindowSnapshot {
	return &WindowSnapshot{
		WindowID:           windowID,
		Epoch:              epoch,
		AccountFree:        amount.NewVector(),
		AccountReserved:    amount.NewVector(),
		PaymentOutstanding: amount.NewVector(),
	}
}

func (s *WindowSnapshot) Clone() *WindowSnapshot {
	if s == nil {
		return nil
	}
	out := *s
	out.AccountFree = s.AccountFree.Clone()
	out.AccountReserved = s.AccountReserved.Clone()
	out.PaymentOutstanding = s.PaymentOutstanding.Clone()
	return &out
}

type Window struct {
	ID                 string
	Asset              string
	Label              string
	OpenEpoch          int
	CloseEpoch         int
	Status             WindowStatus
	Reserve            amount.Amount
	GrossDeposits      amount.Amount
	GrossWithdrawals   amount.Amount
	SettlementVolume   amount.Amount
	AccruedFees        amount.Amount
	DepositsByAccount  amount.Vector
	WithdrawnByAccount amount.Vector
	PostCloseWithdrawn amount.Vector
	Payments           []string
	FeeAdjustments     []FeeAdjustment
	Snapshot           *WindowSnapshot
	Metadata           map[string]string
}

func NewWindow(id, asset string, openEpoch int, closeEpoch int) *Window {
	return &Window{
		ID:                 id,
		Asset:              asset,
		OpenEpoch:          openEpoch,
		CloseEpoch:         closeEpoch,
		Status:             WindowOpen,
		DepositsByAccount:  amount.NewVector(),
		WithdrawnByAccount: amount.NewVector(),
		PostCloseWithdrawn: amount.NewVector(),
		Payments:           []string{},
		FeeAdjustments:     []FeeAdjustment{},
		Metadata:           make(map[string]string),
	}
}

func (w *Window) Clone() *Window {
	out := *w
	out.DepositsByAccount = w.DepositsByAccount.Clone()
	out.WithdrawnByAccount = w.WithdrawnByAccount.Clone()
	out.PostCloseWithdrawn = w.PostCloseWithdrawn.Clone()
	out.Payments = append([]string(nil), w.Payments...)
	out.FeeAdjustments = append([]FeeAdjustment(nil), w.FeeAdjustments...)
	out.Snapshot = w.Snapshot.Clone()
	out.Metadata = make(map[string]string)
	for key, value := range w.Metadata {
		out.Metadata[key] = value
	}
	return &out
}

func (w *Window) IsOpen() bool {
	return w.Status == WindowOpen
}

func (w *Window) IsClosed() bool {
	return w.Status == WindowClosed || w.Status == WindowArchived
}

func (w *Window) AddDeposit(account string, value amount.Amount) error {
	w.Reserve = w.Reserve.MustAdd(value)
	w.GrossDeposits = w.GrossDeposits.MustAdd(value)
	return w.DepositsByAccount.Add(account, value)
}

func (w *Window) AddPayment(paymentID string) {
	for _, existing := range w.Payments {
		if existing == paymentID {
			return
		}
	}
	w.Payments = append(w.Payments, paymentID)
	sort.Strings(w.Payments)
}

func (w *Window) ApplySettlement(value amount.Amount, fee amount.Amount, adjustment FeeAdjustment) error {
	totalDebit := value.MustAdd(fee)
	if totalDebit > w.Reserve {
		return fmt.Errorf("window reserve cannot cover settlement debit")
	}
	w.Reserve = w.Reserve.MustSub(totalDebit)
	w.SettlementVolume = w.SettlementVolume.MustAdd(value)
	w.AccruedFees = w.AccruedFees.MustAdd(fee)
	w.FeeAdjustments = append(w.FeeAdjustments, adjustment)
	return nil
}

func (w *Window) ApplyWithdrawal(account string, value amount.Amount) error {
	if value > w.Reserve {
		return fmt.Errorf("window reserve cannot cover withdrawal")
	}
	w.Reserve = w.Reserve.MustSub(value)
	w.GrossWithdrawals = w.GrossWithdrawals.MustAdd(value)
	if err := w.WithdrawnByAccount.Add(account, value); err != nil {
		return err
	}
	if w.IsClosed() {
		return w.PostCloseWithdrawn.Add(account, value)
	}
	return nil
}

func (w *Window) Outstanding(payments map[string]*Payment) amount.Amount {
	var total amount.Amount
	for _, paymentID := range w.Payments {
		payment := payments[paymentID]
		if payment == nil || !payment.IsOpen() {
			continue
		}
		total = total.MustAdd(payment.Outstanding())
	}
	return total
}

func (w *Window) PaymentCount(payments map[string]*Payment) (open int, closed int) {
	for _, paymentID := range w.Payments {
		payment := payments[paymentID]
		if payment == nil {
			continue
		}
		if payment.IsOpen() {
			open++
		} else {
			closed++
		}
	}
	return open, closed
}

type Event struct {
	Seq       int
	Epoch     int
	Kind      EventKind
	WindowID  string
	AccountID string
	PaymentID string
	Asset     string
	Amount    amount.Amount
	Message   string
	Fields    map[string]string
}

func NewEvent(seq int, epoch int, kind EventKind) Event {
	return Event{
		Seq:    seq,
		Epoch:  epoch,
		Kind:   kind,
		Fields: make(map[string]string),
	}
}

func (e Event) Clone() Event {
	out := e
	out.Fields = make(map[string]string)
	for key, value := range e.Fields {
		out.Fields[key] = value
	}
	return out
}
