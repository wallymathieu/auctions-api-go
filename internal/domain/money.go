package domain

import (
	"fmt"
	"regexp"
	"strconv"
)

// Currency represents a monetary currency
type Currency string

const (
	// VAC is virtual auction currency
	VAC Currency = "VAC"
	// SEK is Swedish Krona
	SEK Currency = "SEK"
	// DKK is Danish Krone
	DKK Currency = "DKK"
)

// Amount represents a monetary amount in a specific currency
type Amount struct {
	Currency Currency `json:"currency"`
	Value    int64    `json:"value"`
}

// String returns a string representation of the amount
func (a Amount) String() string {
	return fmt.Sprintf("%s%d", a.Currency, a.Value)
}

// ParseAmount parses a string into an Amount
func ParseAmount(s string) (*Amount, error) {
	if s == "" {
		return nil, fmt.Errorf("empty amount string")
	}

	// Regex pattern to match currency letter and digit
	// Like "VAC10" or "SEK100"
	pattern := "^([A-Z]+)(\\d+)$"
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(s)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid amount format: %s", s)
	}

	currencyStr := matches[1]
	valueStr := matches[2]

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount value: %s", valueStr)
	}

	return &Amount{
		Currency: Currency(currencyStr),
		Value:    value,
	}, nil
}

// Add adds two amounts together, returning a new amount
// Both amounts must have the same currency
func (a Amount) Add(b Amount) (Amount, error) {
	if a.Currency != b.Currency {
		return Amount{}, fmt.Errorf("cannot add amounts with different currencies: %s and %s", a.Currency, b.Currency)
	}
	return Amount{
		Currency: a.Currency,
		Value:    a.Value + b.Value,
	}, nil
}

// GreaterThan returns true if a is greater than b
func (a Amount) GreaterThan(b Amount) bool {
	return a.Currency == b.Currency && a.Value > b.Value
}

// MarshalJSON implements the json.Marshaler interface
func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a.String())), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (a *Amount) UnmarshalJSON(data []byte) error {
	// Remove quotes
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	parsed, err := ParseAmount(s)
	if err != nil {
		return err
	}

	*a = *parsed
	return nil
}
