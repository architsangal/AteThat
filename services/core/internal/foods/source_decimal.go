package foods

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var sourceNumberPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

// CanonicalNutrientAmount converts provider/Decimal128 notation without a
// binary float or rounding. Bounds prevent enormous exponent allocations.
// User-entered quantities still use the stricter decimal-only validator.
func CanonicalNutrientAmount(raw string) (string, bool) {
	if len(raw) == 0 || len(raw) > 128 || !sourceNumberPattern.MatchString(raw) {
		return "", false
	}
	mantissa, exponent := raw, 0
	if index := strings.IndexAny(raw, "eE"); index >= 0 {
		mantissa = raw[:index]
		value, err := strconv.Atoi(raw[index+1:])
		if err != nil || value < -128 || value > 128 {
			return "", false
		}
		exponent = value
	}
	number, ok := new(big.Rat).SetString(raw)
	if !ok || number.Sign() < 0 {
		return "", false
	}
	if number.Sign() == 0 {
		return "0", true
	}
	places := 0
	if index := strings.IndexByte(mantissa, '.'); index >= 0 {
		places = len(mantissa) - index - 1
	}
	places -= exponent
	if places < 0 {
		places = 0
	}
	result := trimDecimal(number.FloatString(places))
	return result, len(result) <= 32
}
