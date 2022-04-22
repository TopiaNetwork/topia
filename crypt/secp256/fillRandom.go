//go:build !windows

package secp256

/*
#include "./secp256k1/examples/random.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

func fillRandom(random []byte) error {
	if len(random) != 32 {
		return errors.New("input argument error")
	}
	if C.fill_random((*C.uchar)(unsafe.Pointer(&random[0])), C.ulong(uint32(len(random)))) != 1 {
		return errors.New("failed to generate randomness")
	}
	return nil
}
