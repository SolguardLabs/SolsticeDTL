package ids

import (
	"fmt"
	"strings"
	"unicode"
)

type AccountID string
type AssetID string
type WindowID string
type PaymentID string
type EventID string

func Normalize(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}

func Validate(raw string, kind string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fmt.Errorf("%s id is required", kind)
	}
	if len(value) > 64 {
		return fmt.Errorf("%s id is too long", kind)
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '-', '_', ':', '.':
			continue
		default:
			return fmt.Errorf("%s id contains invalid rune %q", kind, r)
		}
	}
	return nil
}

func NewAccountID(raw string) (AccountID, error) {
	value := Normalize(raw)
	if err := Validate(value, "account"); err != nil {
		return "", err
	}
	return AccountID(value), nil
}

func NewAssetID(raw string) (AssetID, error) {
	value := Normalize(raw)
	if err := Validate(value, "asset"); err != nil {
		return "", err
	}
	return AssetID(value), nil
}

func NewWindowID(raw string) (WindowID, error) {
	value := Normalize(raw)
	if err := Validate(value, "window"); err != nil {
		return "", err
	}
	return WindowID(value), nil
}

func NewPaymentID(raw string) (PaymentID, error) {
	value := Normalize(raw)
	if err := Validate(value, "payment"); err != nil {
		return "", err
	}
	return PaymentID(value), nil
}

func NewEventID(raw string) (EventID, error) {
	value := Normalize(raw)
	if err := Validate(value, "event"); err != nil {
		return "", err
	}
	return EventID(value), nil
}

func (id AccountID) String() string {
	return string(id)
}

func (id AssetID) String() string {
	return string(id)
}

func (id WindowID) String() string {
	return string(id)
}

func (id PaymentID) String() string {
	return string(id)
}

func (id EventID) String() string {
	return string(id)
}

func (id AccountID) Empty() bool {
	return strings.TrimSpace(string(id)) == ""
}

func (id AssetID) Empty() bool {
	return strings.TrimSpace(string(id)) == ""
}

func (id WindowID) Empty() bool {
	return strings.TrimSpace(string(id)) == ""
}

func (id PaymentID) Empty() bool {
	return strings.TrimSpace(string(id)) == ""
}

func (id EventID) Empty() bool {
	return strings.TrimSpace(string(id)) == ""
}

func MustAccountID(raw string) AccountID {
	id, err := NewAccountID(raw)
	if err != nil {
		panic(err)
	}
	return id
}

func MustAssetID(raw string) AssetID {
	id, err := NewAssetID(raw)
	if err != nil {
		panic(err)
	}
	return id
}

func MustWindowID(raw string) WindowID {
	id, err := NewWindowID(raw)
	if err != nil {
		panic(err)
	}
	return id
}

func MustPaymentID(raw string) PaymentID {
	id, err := NewPaymentID(raw)
	if err != nil {
		panic(err)
	}
	return id
}

func MustEventID(raw string) EventID {
	id, err := NewEventID(raw)
	if err != nil {
		panic(err)
	}
	return id
}
