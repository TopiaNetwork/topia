package common

import (
	"errors"
	"fmt"
)

var (
	// ErrTransactionClosed is returned when a closed or written transaction is used.
	ErrTransactionClosed = errors.New("transaction has been written or closed")

	// ErrKeyEmpty is returned when attempting to use an empty or nil key.
	ErrKeyEmpty = errors.New("key cannot be empty")

	// ErrValueNil is returned when attempting to set a nil value.
	ErrValueNil = errors.New("value cannot be nil")

	// ErrVersionDoesNotExist is returned when a DB version does not exist.
	ErrVersionDoesNotExist = errors.New("version does not exist")

	// ErrOpenTransactions is returned when open transactions exist which must
	// be discarded/committed before an operation can complete.
	ErrOpenTransactions = errors.New("open transactions exist")

	// ErrReadOnly is returned when a write operation is attempted on a read-only transaction.
	ErrReadOnly = errors.New("cannot modify read-only transaction")

	// ErrInvalidVersion is returned when an operation attempts to use an invalid version ID.
	ErrInvalidVersion = errors.New("invalid version")
)

func ValidateKv(key, value []byte) error {
	if len(key) == 0 {
		return ErrKeyEmpty
	}
	if value == nil {
		return ErrValueNil
	}
	return nil
}

func CombineErrors(ret error, also error, desc string) error {
	if also != nil {
		if ret != nil {
			ret = fmt.Errorf("%w; %s: %v", ret, desc, also)
		} else {
			ret = also
		}
	}
	return ret
}
