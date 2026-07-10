package policy

import (
	"fmt"

	"solsticedtl/src/amount"
)

type SettlementMode string

const (
	ModeSolarNet SettlementMode = "solar_net"
	ModeStrict   SettlementMode = "strict"
)

type Policy struct {
	Name                     string         `json:"name"`
	Mode                     SettlementMode `json:"mode"`
	SettlementFeeBps         int64          `json:"settlementFeeBps"`
	MinimumSettlementFee     amount.Amount  `json:"minimumSettlementFee"`
	MaximumSettlementFee     amount.Amount  `json:"maximumSettlementFee"`
	MinimumDeposit           amount.Amount  `json:"minimumDeposit"`
	MinimumWithdrawal        amount.Amount  `json:"minimumWithdrawal"`
	MinimumPayment           amount.Amount  `json:"minimumPayment"`
	MaxWindowReserve         amount.Amount  `json:"maxWindowReserve"`
	MaxPaymentAmount         amount.Amount  `json:"maxPaymentAmount"`
	MaxPartialSettlementBps  int64          `json:"maxPartialSettlementBps"`
	CloseRequiresSolvency    bool           `json:"closeRequiresSolvency"`
	AllowPostCloseWithdrawal bool           `json:"allowPostCloseWithdrawal"`
}

func Default() Policy {
	return Policy{
		Name:                     "solstice-mainnet-sim",
		Mode:                     ModeSolarNet,
		SettlementFeeBps:         300,
		MinimumSettlementFee:     amount.Must(1),
		MaximumSettlementFee:     amount.Must(50_000),
		MinimumDeposit:           amount.Must(10),
		MinimumWithdrawal:        amount.Must(1),
		MinimumPayment:           amount.Must(1),
		MaxWindowReserve:         amount.Must(10_000_000_000),
		MaxPaymentAmount:         amount.Must(5_000_000_000),
		MaxPartialSettlementBps:  10_000,
		CloseRequiresSolvency:    true,
		AllowPostCloseWithdrawal: true,
	}
}

func (p Policy) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if p.SettlementFeeBps < 0 || p.SettlementFeeBps > amount.BasisPoints {
		return fmt.Errorf("settlement fee bps out of range")
	}
	if p.MaxPartialSettlementBps <= 0 || p.MaxPartialSettlementBps > amount.BasisPoints {
		return fmt.Errorf("partial settlement bps out of range")
	}
	if p.MaximumSettlementFee > 0 && p.MinimumSettlementFee > p.MaximumSettlementFee {
		return fmt.Errorf("minimum settlement fee exceeds maximum")
	}
	if p.MaxPaymentAmount == 0 {
		return fmt.Errorf("max payment amount is required")
	}
	if p.MaxWindowReserve == 0 {
		return fmt.Errorf("max window reserve is required")
	}
	return nil
}

func (p Policy) SettlementFee(gross amount.Amount) (amount.Amount, error) {
	if gross == 0 || p.SettlementFeeBps == 0 {
		return amount.Zero, nil
	}
	fee, err := gross.MulBpsCeil(p.SettlementFeeBps)
	if err != nil {
		return amount.Zero, err
	}
	if fee < p.MinimumSettlementFee {
		fee = p.MinimumSettlementFee
	}
	if p.MaximumSettlementFee > 0 && fee > p.MaximumSettlementFee {
		fee = p.MaximumSettlementFee
	}
	if fee > gross {
		return amount.Zero, fmt.Errorf("settlement fee exceeds gross amount")
	}
	return fee, nil
}

func (p Policy) MaxPartialFor(outstanding amount.Amount) (amount.Amount, error) {
	if p.MaxPartialSettlementBps == amount.BasisPoints {
		return outstanding, nil
	}
	return outstanding.MulBpsFloor(p.MaxPartialSettlementBps)
}

func (p Policy) CheckDeposit(value amount.Amount) error {
	if value < p.MinimumDeposit {
		return fmt.Errorf("deposit below minimum")
	}
	return nil
}

func (p Policy) CheckWithdrawal(value amount.Amount) error {
	if value < p.MinimumWithdrawal {
		return fmt.Errorf("withdrawal below minimum")
	}
	return nil
}

func (p Policy) CheckPayment(value amount.Amount) error {
	if value < p.MinimumPayment {
		return fmt.Errorf("payment below minimum")
	}
	if value > p.MaxPaymentAmount {
		return fmt.Errorf("payment exceeds maximum")
	}
	return nil
}

func (p Policy) CheckWindowReserve(current amount.Amount, add amount.Amount) error {
	next := current.MustAdd(add)
	if next > p.MaxWindowReserve {
		return fmt.Errorf("window reserve exceeds policy cap")
	}
	return nil
}

func (p Policy) WithFeeBps(bps int64) Policy {
	p.SettlementFeeBps = bps
	return p
}
