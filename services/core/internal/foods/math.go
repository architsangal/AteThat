package foods

import (
	"errors"
	"math/big"
	"strings"
)

// Scale uses rational arithmetic so a serving never passes through a binary
// float. Six decimal places are retained at the app's nutrition boundary.
func Scale(amount, quantity, base string) (string, error) {
	parse := func(raw string) (*big.Rat, bool) {
		if !decimalString(raw) {
			return nil, false
		}
		return new(big.Rat).SetString(raw)
	}
	a, okA := parse(amount)
	q, okQ := parse(quantity)
	b, okB := parse(base)
	if !okA || !okQ || !okB || a.Sign() < 0 || q.Sign() <= 0 || b.Sign() <= 0 {
		return "", errors.New("invalid nutrition quantity")
	}
	result := new(big.Rat).Quo(new(big.Rat).Mul(a, q), b)
	return trimDecimal(result.FloatString(6)), nil
}

func Sum(values []string) (string, error) {
	total := new(big.Rat)
	for _, raw := range values {
		if !decimalString(raw) {
			return "", errors.New("invalid nutrition value")
		}
		value, ok := new(big.Rat).SetString(raw)
		if !ok || value.Sign() < 0 {
			return "", errors.New("invalid nutrition value")
		}
		total.Add(total, value)
	}
	return trimDecimal(total.FloatString(6)), nil
}

func decimalString(raw string) bool {
	if len(raw) == 0 || len(raw) > 32 || raw[0] < '0' || raw[0] > '9' {
		return false
	}
	dot := false
	for i := 1; i < len(raw); i++ {
		if raw[i] == '.' && !dot && i < len(raw)-1 {
			dot = true
			continue
		}
		if raw[i] < '0' || raw[i] > '9' {
			return false
		}
	}
	return true
}

func trimDecimal(raw string) string {
	if strings.Contains(raw, ".") {
		raw = strings.TrimRight(strings.TrimRight(raw, "0"), ".")
	}
	return raw
}
