package common

import (
	"fmt"
	"math/big"
)

func SafeMul(a uint64, b uint64) *big.Int {
	return big.NewInt(0).Mul(big.NewInt(0).SetUint64(a), big.NewInt(0).SetUint64(b))
}

func SafeSubUint64(a, b uint64) (uint64, error) {
	if a < b {
		return 0, fmt.Errorf("Substrate overflow of a %d - b %d", a, b)
	}
	return a - b, nil
}

func SafeAddUint64(a, b uint64) (uint64, error) {
	s := a + b
	if s >= a && s >= b {
		return s, nil
	}
	return 0, fmt.Errorf("Add overflow of a %d - b %d", a, b)
}
