package scenario

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"solsticedtl/src/amount"
	"solsticedtl/src/domain"
	"solsticedtl/src/engine"
	"solsticedtl/src/policy"
)

type PolicyConfig struct {
	Name                     *string        `json:"name"`
	Mode                     *string        `json:"mode"`
	SettlementFeeBps         *int64         `json:"settlementFeeBps"`
	MinimumSettlementFee     *amount.Amount `json:"minimumSettlementFee"`
	MaximumSettlementFee     *amount.Amount `json:"maximumSettlementFee"`
	MinimumDeposit           *amount.Amount `json:"minimumDeposit"`
	MinimumWithdrawal        *amount.Amount `json:"minimumWithdrawal"`
	MinimumPayment           *amount.Amount `json:"minimumPayment"`
	MaxWindowReserve         *amount.Amount `json:"maxWindowReserve"`
	MaxPaymentAmount         *amount.Amount `json:"maxPaymentAmount"`
	MaxPartialSettlementBps  *int64         `json:"maxPartialSettlementBps"`
	CloseRequiresSolvency    *bool          `json:"closeRequiresSolvency"`
	AllowPostCloseWithdrawal *bool          `json:"allowPostCloseWithdrawal"`
}

func (c PolicyConfig) Apply(base policy.Policy) policy.Policy {
	if c.Name != nil {
		base.Name = *c.Name
	}
	if c.Mode != nil {
		base.Mode = policy.SettlementMode(*c.Mode)
	}
	if c.SettlementFeeBps != nil {
		base.SettlementFeeBps = *c.SettlementFeeBps
	}
	if c.MinimumSettlementFee != nil {
		base.MinimumSettlementFee = *c.MinimumSettlementFee
	}
	if c.MaximumSettlementFee != nil {
		base.MaximumSettlementFee = *c.MaximumSettlementFee
	}
	if c.MinimumDeposit != nil {
		base.MinimumDeposit = *c.MinimumDeposit
	}
	if c.MinimumWithdrawal != nil {
		base.MinimumWithdrawal = *c.MinimumWithdrawal
	}
	if c.MinimumPayment != nil {
		base.MinimumPayment = *c.MinimumPayment
	}
	if c.MaxWindowReserve != nil {
		base.MaxWindowReserve = *c.MaxWindowReserve
	}
	if c.MaxPaymentAmount != nil {
		base.MaxPaymentAmount = *c.MaxPaymentAmount
	}
	if c.MaxPartialSettlementBps != nil {
		base.MaxPartialSettlementBps = *c.MaxPartialSettlementBps
	}
	if c.CloseRequiresSolvency != nil {
		base.CloseRequiresSolvency = *c.CloseRequiresSolvency
	}
	if c.AllowPostCloseWithdrawal != nil {
		base.AllowPostCloseWithdrawal = *c.AllowPostCloseWithdrawal
	}
	return base
}

type Step struct {
	Action string `json:"action"`

	ID          string            `json:"id"`
	AccountID   string            `json:"accountId"`
	Role        string            `json:"role"`
	DisplayName string            `json:"displayName"`
	Source      string            `json:"source"`
	Beneficiary string            `json:"beneficiary"`
	Operator    string            `json:"operator"`
	PaymentID   string            `json:"paymentId"`
	WindowID    string            `json:"windowId"`
	Asset       string            `json:"asset"`
	Symbol      string            `json:"symbol"`
	Label       string            `json:"label"`
	Decimals    int               `json:"decimals"`
	OpenEpoch   int               `json:"openEpoch"`
	CloseEpoch  int               `json:"closeEpoch"`
	DueEpoch    int               `json:"dueEpoch"`
	Priority    int               `json:"priority"`
	Epochs      int               `json:"epochs"`
	To          int               `json:"to"`
	Amount      amount.Amount     `json:"amount"`
	Metadata    map[string]string `json:"metadata"`
}

type Scenario struct {
	Name     string                  `json:"name"`
	Policy   PolicyConfig            `json:"policy"`
	Assets   []engine.AssetRequest   `json:"assets"`
	Accounts []engine.AccountRequest `json:"accounts"`
	Windows  []engine.WindowRequest  `json:"windows"`
	Steps    []Step                  `json:"steps"`
}

type Result struct {
	Service *engine.Service
	Report  domain.PoolReport
}

func LoadFile(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var scenario Scenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, err
	}
	if strings.TrimSpace(scenario.Name) == "" {
		scenario.Name = "SolsticeDTL"
	}
	return &scenario, nil
}

func RunFile(path string, includeEvents bool) (*Result, error) {
	loaded, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	return loaded.Run(includeEvents)
}

func (s Scenario) Run(includeEvents bool) (*Result, error) {
	rules := s.Policy.Apply(policy.Default())
	service, err := engine.New(s.Name, rules)
	if err != nil {
		return nil, err
	}
	for _, asset := range s.Assets {
		if err := service.RegisterAsset(asset); err != nil {
			return nil, fmt.Errorf("asset %s: %w", asset.ID, err)
		}
	}
	for _, account := range s.Accounts {
		if err := service.RegisterAccount(account); err != nil {
			return nil, fmt.Errorf("account %s: %w", account.ID, err)
		}
	}
	for _, window := range s.Windows {
		if err := service.OpenWindow(window); err != nil {
			return nil, fmt.Errorf("window %s: %w", window.ID, err)
		}
	}
	for index, step := range s.Steps {
		if err := executeStep(service, step); err != nil {
			return nil, fmt.Errorf("step %d %s: %w", index+1, step.Action, err)
		}
	}
	return &Result{Service: service, Report: service.Report(includeEvents)}, nil
}

func executeStep(service *engine.Service, step Step) error {
	switch strings.ToLower(strings.TrimSpace(step.Action)) {
	case "account", "create_account", "register_account":
		id := firstNonEmpty(step.ID, step.AccountID)
		return service.RegisterAccount(engine.AccountRequest{
			ID:          id,
			Role:        step.Role,
			DisplayName: step.DisplayName,
			Metadata:    step.Metadata,
		})
	case "asset", "list_asset", "register_asset":
		return service.RegisterAsset(engine.AssetRequest{
			ID:                 step.ID,
			Symbol:             step.Symbol,
			Decimals:           step.Decimals,
			SettlementPriority: step.Priority,
			Metadata:           step.Metadata,
		})
	case "window", "open_window":
		id := firstNonEmpty(step.ID, step.WindowID)
		return service.OpenWindow(engine.WindowRequest{
			ID:         id,
			Asset:      step.Asset,
			Label:      step.Label,
			OpenEpoch:  step.OpenEpoch,
			CloseEpoch: step.CloseEpoch,
			Metadata:   step.Metadata,
		})
	case "advance", "advance_epoch":
		return service.Advance(engine.AdvanceRequest{Epochs: step.Epochs, To: step.To})
	case "deposit":
		return service.Deposit(engine.DepositRequest{
			AccountID: step.AccountID,
			WindowID:  step.WindowID,
			Asset:     step.Asset,
			Amount:    step.Amount,
		})
	case "queue", "queue_payment", "payment":
		id := firstNonEmpty(step.ID, step.PaymentID)
		return service.QueuePayment(engine.QueuePaymentRequest{
			ID:          id,
			WindowID:    step.WindowID,
			Source:      step.Source,
			Beneficiary: step.Beneficiary,
			Asset:       step.Asset,
			Amount:      step.Amount,
			DueEpoch:    step.DueEpoch,
			Priority:    step.Priority,
			Metadata:    step.Metadata,
		})
	case "close", "close_window":
		id := firstNonEmpty(step.ID, step.WindowID)
		return service.CloseWindow(engine.CloseWindowRequest{WindowID: id})
	case "settle", "settle_payment":
		id := firstNonEmpty(step.ID, step.PaymentID)
		return service.Settle(engine.SettlementRequest{
			PaymentID: id,
			WindowID:  step.WindowID,
			Amount:    step.Amount,
			Operator:  step.Operator,
		})
	case "withdraw", "withdrawal":
		return service.Withdraw(engine.WithdrawalRequest{
			AccountID: step.AccountID,
			WindowID:  step.WindowID,
			Asset:     step.Asset,
			Amount:    step.Amount,
		})
	case "validate":
		service.Validate()
		return nil
	default:
		return fmt.Errorf("unknown action %q", step.Action)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ValidateFile(path string) ([]string, error) {
	result, err := RunFile(path, false)
	if err != nil {
		return nil, err
	}
	return result.Service.Validate(), nil
}

func EncodeReport(report domain.PoolReport, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(report, "", "  ")
	}
	return json.Marshal(report)
}
