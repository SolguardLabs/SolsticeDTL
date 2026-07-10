package ledger

import (
	"fmt"
	"sort"
	"strings"

	"solsticedtl/src/amount"
	"solsticedtl/src/domain"
)

type Direction string

const (
	Debit  Direction = "debit"
	Credit Direction = "credit"
)

type EntryKind string

const (
	EntryDeposit       EntryKind = "deposit"
	EntryReserve       EntryKind = "reserve"
	EntrySettlement    EntryKind = "settlement"
	EntryFee           EntryKind = "fee"
	EntryWithdrawal    EntryKind = "withdrawal"
	EntryWindowReserve EntryKind = "window_reserve"
)

type Entry struct {
	Seq       int
	Epoch     int
	Kind      EntryKind
	Direction Direction
	AccountID string
	WindowID  string
	PaymentID string
	Asset     string
	Amount    amount.Amount
	Memo      string
}

type Journal struct {
	entries []Entry
	index   map[string][]int
}

func NewJournal() *Journal {
	return &Journal{
		entries: []Entry{},
		index:   make(map[string][]int),
	}
}

func (j *Journal) Clone() *Journal {
	out := NewJournal()
	out.entries = append(out.entries, j.entries...)
	for key, values := range j.index {
		out.index[key] = append([]int(nil), values...)
	}
	return out
}

func (j *Journal) Append(entry Entry) Entry {
	entry.Seq = len(j.entries) + 1
	j.entries = append(j.entries, entry)
	for _, key := range entryKeys(entry) {
		j.index[key] = append(j.index[key], entry.Seq-1)
	}
	return entry
}

func (j *Journal) Entries() []Entry {
	return append([]Entry(nil), j.entries...)
}

func (j *Journal) ForAccount(accountID string) []Entry {
	return j.byKey("account:" + accountID)
}

func (j *Journal) ForWindow(windowID string) []Entry {
	return j.byKey("window:" + windowID)
}

func (j *Journal) ForPayment(paymentID string) []Entry {
	return j.byKey("payment:" + paymentID)
}

func (j *Journal) ForAsset(asset string) []Entry {
	return j.byKey("asset:" + asset)
}

func (j *Journal) byKey(key string) []Entry {
	offsets := j.index[key]
	out := make([]Entry, 0, len(offsets))
	for _, offset := range offsets {
		if offset >= 0 && offset < len(j.entries) {
			out = append(out, j.entries[offset])
		}
	}
	return out
}

func (j *Journal) TrialBalance(asset string) amount.Amount {
	var credits amount.Amount
	var debits amount.Amount
	for _, entry := range j.entries {
		if entry.Asset != asset {
			continue
		}
		switch entry.Direction {
		case Credit:
			credits = credits.MustAdd(entry.Amount)
		case Debit:
			debits = debits.MustAdd(entry.Amount)
		}
	}
	if credits >= debits {
		return credits.MustSub(debits)
	}
	return amount.Zero
}

func entryKeys(entry Entry) []string {
	keys := []string{"asset:" + entry.Asset}
	if entry.AccountID != "" {
		keys = append(keys, "account:"+entry.AccountID)
	}
	if entry.WindowID != "" {
		keys = append(keys, "window:"+entry.WindowID)
	}
	if entry.PaymentID != "" {
		keys = append(keys, "payment:"+entry.PaymentID)
	}
	return keys
}

func Deposit(epoch int, account *domain.Account, window *domain.Window, asset string, value amount.Amount) ([]Entry, error) {
	if account == nil || window == nil {
		return nil, fmt.Errorf("deposit requires account and window")
	}
	if err := account.Deposit(asset, value); err != nil {
		return nil, err
	}
	if err := window.AddDeposit(account.ID, value); err != nil {
		return nil, err
	}
	return []Entry{
		{
			Epoch:     epoch,
			Kind:      EntryDeposit,
			Direction: Credit,
			AccountID: account.ID,
			WindowID:  window.ID,
			Asset:     asset,
			Amount:    value,
			Memo:      "account free balance credited",
		},
		{
			Epoch:     epoch,
			Kind:      EntryWindowReserve,
			Direction: Credit,
			AccountID: account.ID,
			WindowID:  window.ID,
			Asset:     asset,
			Amount:    value,
			Memo:      "window reserve increased",
		},
	}, nil
}

func QueuePayment(epoch int, account *domain.Account, payment *domain.Payment) ([]Entry, error) {
	if account == nil || payment == nil {
		return nil, fmt.Errorf("queue payment requires account and payment")
	}
	if err := account.Reserve(payment.Asset, payment.Amount); err != nil {
		return nil, err
	}
	return []Entry{
		{
			Epoch:     epoch,
			Kind:      EntryReserve,
			Direction: Debit,
			AccountID: account.ID,
			WindowID:  payment.WindowID,
			PaymentID: payment.ID,
			Asset:     payment.Asset,
			Amount:    payment.Amount,
			Memo:      "free balance moved to reserve",
		},
		{
			Epoch:     epoch,
			Kind:      EntryReserve,
			Direction: Credit,
			AccountID: account.ID,
			WindowID:  payment.WindowID,
			PaymentID: payment.ID,
			Asset:     payment.Asset,
			Amount:    payment.Amount,
			Memo:      "reserved payment backing recorded",
		},
	}, nil
}

func SettlePayment(epoch int, source *domain.Account, beneficiary *domain.Account, window *domain.Window, payment *domain.Payment, value amount.Amount, fee amount.Amount, operator string) ([]Entry, domain.SettlementTranche, error) {
	if source == nil || beneficiary == nil || window == nil || payment == nil {
		return nil, domain.SettlementTranche{}, fmt.Errorf("settlement requires source, beneficiary, window and payment")
	}
	tranche, err := payment.ApplySettlement(epoch, value, fee, operator)
	if err != nil {
		return nil, domain.SettlementTranche{}, err
	}
	if err := source.ConsumeReserved(payment.Asset, value); err != nil {
		return nil, domain.SettlementTranche{}, err
	}
	if err := source.PayFee(payment.Asset, fee); err != nil {
		return nil, domain.SettlementTranche{}, err
	}
	if err := beneficiary.ReceiveSettlement(payment.Asset, tranche.Net); err != nil {
		return nil, domain.SettlementTranche{}, err
	}
	adjustment := domain.FeeAdjustment{
		Epoch:     epoch,
		Asset:     payment.Asset,
		Basis:     value,
		Fee:       fee,
		Reason:    "settlement_service_charge",
		PaymentID: payment.ID,
	}
	if err := window.ApplySettlement(value, fee, adjustment); err != nil {
		return nil, domain.SettlementTranche{}, err
	}
	entries := []Entry{
		{
			Epoch:     epoch,
			Kind:      EntrySettlement,
			Direction: Debit,
			AccountID: source.ID,
			WindowID:  window.ID,
			PaymentID: payment.ID,
			Asset:     payment.Asset,
			Amount:    value,
			Memo:      "reserved payment backing consumed",
		},
		{
			Epoch:     epoch,
			Kind:      EntrySettlement,
			Direction: Credit,
			AccountID: beneficiary.ID,
			WindowID:  window.ID,
			PaymentID: payment.ID,
			Asset:     payment.Asset,
			Amount:    tranche.Net,
			Memo:      "beneficiary net settlement credited",
		},
		{
			Epoch:     epoch,
			Kind:      EntryFee,
			Direction: Credit,
			AccountID: source.ID,
			WindowID:  window.ID,
			PaymentID: payment.ID,
			Asset:     payment.Asset,
			Amount:    fee,
			Memo:      "settlement fee recorded",
		},
		{
			Epoch:     epoch,
			Kind:      EntryWindowReserve,
			Direction: Debit,
			AccountID: source.ID,
			WindowID:  window.ID,
			PaymentID: payment.ID,
			Asset:     payment.Asset,
			Amount:    value.MustAdd(fee),
			Memo:      "window reserve debited for settlement and fees",
		},
	}
	return entries, tranche, nil
}

func Withdraw(epoch int, account *domain.Account, window *domain.Window, asset string, value amount.Amount) ([]Entry, error) {
	if account == nil || window == nil {
		return nil, fmt.Errorf("withdrawal requires account and window")
	}
	if err := account.Withdraw(asset, value); err != nil {
		return nil, err
	}
	if err := window.ApplyWithdrawal(account.ID, value); err != nil {
		return nil, err
	}
	return []Entry{
		{
			Epoch:     epoch,
			Kind:      EntryWithdrawal,
			Direction: Debit,
			AccountID: account.ID,
			WindowID:  window.ID,
			Asset:     asset,
			Amount:    value,
			Memo:      "account free balance withdrawn",
		},
		{
			Epoch:     epoch,
			Kind:      EntryWindowReserve,
			Direction: Debit,
			AccountID: account.ID,
			WindowID:  window.ID,
			Asset:     asset,
			Amount:    value,
			Memo:      "window reserve released",
		},
	}, nil
}

func AppendAll(journal *Journal, entries []Entry) {
	for _, entry := range entries {
		journal.Append(entry)
	}
}

type AccountLedgerReport struct {
	AccountID string
	Asset     string
	Credits   amount.Amount
	Debits    amount.Amount
	Net       amount.Amount
	Lines     int
}

func AccountReports(entries []Entry) []AccountLedgerReport {
	type key struct {
		account string
		asset   string
	}
	buckets := make(map[key]*AccountLedgerReport)
	for _, entry := range entries {
		if entry.AccountID == "" {
			continue
		}
		k := key{account: entry.AccountID, asset: entry.Asset}
		report := buckets[k]
		if report == nil {
			report = &AccountLedgerReport{AccountID: entry.AccountID, Asset: entry.Asset}
			buckets[k] = report
		}
		report.Lines++
		if entry.Direction == Credit {
			report.Credits = report.Credits.MustAdd(entry.Amount)
		} else {
			report.Debits = report.Debits.MustAdd(entry.Amount)
		}
	}
	out := make([]AccountLedgerReport, 0, len(buckets))
	for _, report := range buckets {
		if report.Credits >= report.Debits {
			report.Net = report.Credits.MustSub(report.Debits)
		}
		out = append(out, *report)
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].AccountID, out[i].Asset}, ":")
		right := strings.Join([]string{out[j].AccountID, out[j].Asset}, ":")
		return left < right
	})
	return out
}
