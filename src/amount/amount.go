package amount

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Amount int64

const (
	Zero Amount = 0
	One  Amount = 1

	BasisPoints int64 = 10_000
)

var (
	ErrNegativeAmount = errors.New("amount cannot be negative")
	ErrOverflow       = errors.New("amount overflow")
	ErrDivideByZero   = errors.New("amount division by zero")
)

func New(value int64) (Amount, error) {
	if value < 0 {
		return Zero, ErrNegativeAmount
	}
	return Amount(value), nil
}

func Must(value int64) Amount {
	amount, err := New(value)
	if err != nil {
		panic(err)
	}
	return amount
}

func Parse(value string) (Amount, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Zero, fmt.Errorf("empty amount")
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return Zero, err
	}
	return New(parsed)
}

func (a Amount) Int64() int64 {
	return int64(a)
}

func (a Amount) IsZero() bool {
	return a == Zero
}

func (a Amount) IsPositive() bool {
	return a > Zero
}

func (a Amount) String() string {
	return strconv.FormatInt(int64(a), 10)
}

func (a Amount) Add(b Amount) (Amount, error) {
	if b < 0 || a < 0 {
		return Zero, ErrNegativeAmount
	}
	if int64(a) > math.MaxInt64-int64(b) {
		return Zero, ErrOverflow
	}
	return a + b, nil
}

func (a Amount) MustAdd(b Amount) Amount {
	value, err := a.Add(b)
	if err != nil {
		panic(err)
	}
	return value
}

func (a Amount) Sub(b Amount) (Amount, error) {
	if b < 0 || a < 0 {
		return Zero, ErrNegativeAmount
	}
	if b > a {
		return Zero, ErrNegativeAmount
	}
	return a - b, nil
}

func (a Amount) MustSub(b Amount) Amount {
	value, err := a.Sub(b)
	if err != nil {
		panic(err)
	}
	return value
}

func (a Amount) MulScalar(multiplier int64) (Amount, error) {
	if multiplier < 0 || a < 0 {
		return Zero, ErrNegativeAmount
	}
	if a != 0 && multiplier > math.MaxInt64/int64(a) {
		return Zero, ErrOverflow
	}
	return Amount(int64(a) * multiplier), nil
}

func (a Amount) MulBpsFloor(bps int64) (Amount, error) {
	if bps < 0 {
		return Zero, ErrNegativeAmount
	}
	if a == 0 || bps == 0 {
		return Zero, nil
	}
	product := int64(a) * bps
	if int64(a) != product/bps {
		return Zero, ErrOverflow
	}
	return Amount(product / BasisPoints), nil
}

func (a Amount) MulBpsCeil(bps int64) (Amount, error) {
	if bps < 0 {
		return Zero, ErrNegativeAmount
	}
	if a == 0 || bps == 0 {
		return Zero, nil
	}
	product := int64(a) * bps
	if int64(a) != product/bps {
		return Zero, ErrOverflow
	}
	return Amount((product + BasisPoints - 1) / BasisPoints), nil
}

func (a Amount) RatioFloor(numerator, denominator int64) (Amount, error) {
	if numerator < 0 || denominator < 0 || a < 0 {
		return Zero, ErrNegativeAmount
	}
	if denominator == 0 {
		return Zero, ErrDivideByZero
	}
	if a == 0 || numerator == 0 {
		return Zero, nil
	}
	product := int64(a) * numerator
	if int64(a) != product/numerator {
		return Zero, ErrOverflow
	}
	return Amount(product / denominator), nil
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(a), 10)), nil
}

func (a *Amount) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch value := raw.(type) {
	case float64:
		if value < 0 || math.Trunc(value) != value {
			return fmt.Errorf("invalid amount %v", value)
		}
		*a = Amount(int64(value))
		return nil
	case string:
		parsed, err := Parse(value)
		if err != nil {
			return err
		}
		*a = parsed
		return nil
	default:
		return fmt.Errorf("invalid amount type %T", raw)
	}
}

func Min(values ...Amount) Amount {
	if len(values) == 0 {
		return Zero
	}
	best := values[0]
	for _, value := range values[1:] {
		if value < best {
			best = value
		}
	}
	return best
}

func Max(values ...Amount) Amount {
	if len(values) == 0 {
		return Zero
	}
	best := values[0]
	for _, value := range values[1:] {
		if value > best {
			best = value
		}
	}
	return best
}

func Clamp(value, lower, upper Amount) Amount {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}

type Vector map[string]Amount

func NewVector() Vector {
	return make(Vector)
}

func (v Vector) Clone() Vector {
	out := NewVector()
	for key, value := range v {
		out[key] = value
	}
	return out
}

func (v Vector) Get(key string) Amount {
	return v[key]
}

func (v Vector) Add(key string, value Amount) error {
	if value < 0 {
		return ErrNegativeAmount
	}
	next, err := v[key].Add(value)
	if err != nil {
		return err
	}
	v[key] = next
	return nil
}

func (v Vector) Sub(key string, value Amount) error {
	if value < 0 {
		return ErrNegativeAmount
	}
	next, err := v[key].Sub(value)
	if err != nil {
		return err
	}
	v[key] = next
	return nil
}

func (v Vector) Sum() Amount {
	var total Amount
	for _, value := range v {
		total = total.MustAdd(value)
	}
	return total
}

func (v Vector) PositiveEntries() map[string]Amount {
	out := NewVector()
	for key, value := range v {
		if value > 0 {
			out[key] = value
		}
	}
	return out
}
