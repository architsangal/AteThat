package targets

import (
	"errors"
	"fmt"
	"math/big"
)

// LowActivityMaintenance reuses the validated adult Inactive EER branch,
// independently of macro targets and the user's usual-activity setting.
func LowActivityMaintenance(age uint32, sex, height, weight string) (string, string, error) {
	if age < 19 {
		return "", "", errors.New("Automatic maintenance estimates are available from age 19. You can enter a baseline manually.")
	}
	equations, ok := adultEquations[sex]
	if !ok {
		return "", "", errors.New("Add date of birth and equation-specific sex in Profile to calculate an estimate, or enter maintenance calories manually.")
	}
	if !ValidSelectedAmount("ENERGY", height) || !ValidSelectedAmount("ENERGY", weight) {
		return "", "", errors.New("Review your profile height and weight, or enter a baseline manually.")
	}
	c := equations["Inactive"]
	n := func(s string) *big.Rat { r, _ := new(big.Rat).SetString(s); return r }
	total := n(c[0])
	for i, v := range []string{fmt.Sprint(age), height, weight} {
		total.Add(total, new(big.Rat).Mul(n(c[i+1]), n(v)))
	}
	value := decimalText(total)
	if total.Sign() <= 0 || len(value) > 32 {
		return "", "", errors.New("These inputs do not produce a supported estimate. Review your profile or enter a baseline manually.")
	}
	return value, fmt.Sprintf("%s − (%s × age) + (%s × height in cm) + (%s × weight in kg)", c[0], c[1][1:], c[2], c[3]), nil
}
