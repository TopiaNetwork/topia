// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"errors"
	"github.com/TopiaNetwork/topia/codec"
	"math/big"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidSig           = errors.New("invalid transaction v, r, s values")
	ErrUnexpectedProtection = errors.New("transaction type does not supported EIP-155 protected signatures")
	ErrInvalidTxType        = errors.New("transaction type not valid in this context")
	ErrTxTypeNotSupported   = errors.New("transaction type not supported")
	ErrGasFeeCapTooLow      = errors.New("fee cap less than base fee")
	errShortTypedTx         = errors.New("typed transaction too short")
)

// Transaction types.
const (
	LegacyTxType = iota
	AccessListTxType
	DynamicFeeTxType

	HashLength    = 32
	AddressLength = 20
)

type Hash [HashLength]byte
type Address [AddressLength]byte

// Transaction is an Ethereum transaction.
type Transaction struct {
	inner TxData    // Consensus contents of a transaction
	time  time.Time // Time first seen locally (spam avoidance)

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

// NewTx creates a new transaction.
func NewTx(inner TxData) *Transaction {
	tx := new(Transaction)
	tx.setDecoded(inner.copy(), 0)
	return tx
}

// TxData is the underlying data of a transaction.
//
// This is implemented by DynamicFeeTx, LegacyTx and AccessListTx.
type TxData interface {
	txType() byte // returns the type ID
	copy() TxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	accessList() AccessList
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *Address

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)
}

//// EncodeRLP implements rlp.Encoder
//func (tx *Transaction) EncodeRLP(w io.Writer) error {
//	if tx.Type() == LegacyTxType {
//		return rlp.Encode(w, tx.inner)
//	}
//	// It's an EIP-2718 typed TX envelope.
//	buf := encodeBufferPool.Get().(*bytes.Buffer)
//	defer encodeBufferPool.Put(buf)
//	buf.Reset()
//	if err := tx.encodeTyped(buf); err != nil {
//		return err
//	}
//	return rlp.Encode(w, buf.Bytes())
//}

// encodeTyped writes the canonical encoding of a typed transaction to w.
//func (tx *Transaction) encodeTyped(w *bytes.Buffer) error {
//	w.WriteByte(tx.Type())
//	return rlp.Encode(w, tx.inner)
//}

// MarshalBinary returns the canonical encoding of the transaction.
// For legacy transactions, it returns the RLP encoding. For EIP-2718 typed
//// transactions, it returns the type and payload.
//func (tx *Transaction) MarshalBinary() ([]byte, error) {
//	if tx.Type() == LegacyTxType {
//		return rlp.EncodeToBytes(tx.inner)
//	}
//	var buf bytes.Buffer
//	err := tx.encodeTyped(&buf)
//	return buf.Bytes(), err
//}

//// DecodeRLP implements rlp.Decoder
//func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
//	kind, size, err := s.Kind()
//	switch {
//	case err != nil:
//		return err
//	case kind == rlp.List:
//		// It's a legacy transaction.
//		var inner LegacyTx
//		err := s.Decode(&inner)
//		if err == nil {
//			tx.setDecoded(&inner, int(rlp.ListSize(size)))
//		}
//		return err
//	default:
//		// It's an EIP-2718 typed TX envelope.
//		var b []byte
//		if b, err = s.Bytes(); err != nil {
//			return err
//		}
//		inner, err := tx.decodeTyped(b)
//		if err == nil {
//			tx.setDecoded(inner, len(b))
//		}
//		return err
//	}
//}

// UnmarshalBinary decodes the canonical encoding of transactions.
// It supports legacy RLP transactions and EIP2718 typed transactions.
func (tx *Transaction) UnmarshalBinary(b []byte) error {
	if len(b) > 0 && b[0] > 0x7f {
		// It's a legacy transaction.
		var data LegacyTx
		marshaler := codec.CreateMarshaler(codec.CodecType_RLP)
		err := marshaler.Unmarshal(b, &data)
		if err != nil {
			return err
		}
		tx.setDecoded(&data, len(b))
		return nil
	}
	// It's an EIP2718 typed transaction envelope.
	inner, err := tx.decodeTyped(b)
	if err != nil {
		return err
	}
	tx.setDecoded(inner, len(b))
	return nil
}

// decodeTyped decodes a typed transaction from the canonical format.
func (tx *Transaction) decodeTyped(b []byte) (TxData, error) {
	if len(b) <= 1 {
		return nil, errShortTypedTx
	}
	switch b[0] {
	case AccessListTxType:
		var inner AccessListTx
		marshaler := codec.CreateMarshaler(codec.CodecType_RLP)
		err := marshaler.Unmarshal(b[1:], &inner)
		return &inner, err
	case DynamicFeeTxType:
		var inner DynamicFeeTx
		marshaler := codec.CreateMarshaler(codec.CodecType_RLP)
		err := marshaler.Unmarshal(b[1:], &inner)
		return &inner, err
	default:
		return nil, ErrTxTypeNotSupported
	}
}

// setDecoded sets the inner transaction and size after decoding.
func (tx *Transaction) setDecoded(inner TxData, size int) {
	tx.inner = inner
	tx.time = time.Now()
	if size > 0 {
		tx.size.Store(StorageSize(size))
	}
}

//func sanityCheckSignature(v *big.Int, r *big.Int, s *big.Int, maybeProtected bool) error {
//	if isProtectedV(v) && !maybeProtected {
//		return ErrUnexpectedProtection
//	}
//
//	var plainV byte
//	if isProtectedV(v) {
//		chainID := deriveChainId(v).Uint64()
//		plainV = byte(v.Uint64() - 35 - 2*chainID)
//	} else if maybeProtected {
//		// Only EIP-155 signatures can be optionally protected. Since
//		// we determined this v value is not protected, it must be a
//		// raw 27 or 28.
//		plainV = byte(v.Uint64() - 27)
//	} else {
//		// If the signature is not optionally protected, we assume it
//		// must already be equal to the recovery id.
//		plainV = byte(v.Uint64())
//	}
//	if !crypto.ValidateSignatureValues(plainV, r, s, false) {
//		return ErrInvalidSig
//	}
//
//	return nil
//}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// Protected says whether the transaction is replay-protected.
func (tx *Transaction) Protected() bool {
	switch tx := tx.inner.(type) {
	case *LegacyTx:
		return tx.V != nil && isProtectedV(tx.V)
	default:
		return true
	}
}

// Type returns the transaction type.
func (tx *Transaction) Type() uint8 {
	return tx.inner.txType()
}

// ChainId returns the EIP155 chain ID of the transaction. The return value will always be
// non-nil. For legacy transactions which are not replay-protected, the return value is
// zero.
func (tx *Transaction) ChainId() *big.Int {
	return tx.inner.chainID()
}

// Data returns the input data of the transaction.
func (tx *Transaction) Data() []byte { return tx.inner.data() }

// AccessList returns the access list of the transaction.
func (tx *Transaction) AccessList() AccessList { return tx.inner.accessList() }

// Gas returns the gas limit of the transaction.
func (tx *Transaction) Gas() uint64 { return tx.inner.gas() }

// GasPrice returns the gas price of the transaction.
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.inner.gasPrice()) }

// GasTipCap returns the gasTipCap per gas of the transaction.
func (tx *Transaction) GasTipCap() *big.Int { return new(big.Int).Set(tx.inner.gasTipCap()) }

// GasFeeCap returns the fee cap per gas of the transaction.
func (tx *Transaction) GasFeeCap() *big.Int { return new(big.Int).Set(tx.inner.gasFeeCap()) }

// Value returns the ether amount of the transaction.
func (tx *Transaction) Value() *big.Int { return new(big.Int).Set(tx.inner.value()) }

// Nonce returns the sender account nonce of the transaction.
func (tx *Transaction) Nonce() uint64 { return tx.inner.nonce() }

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *Transaction) To() *Address {
	return copyAddressPtr(tx.inner.to())
}

// Cost returns gas * gasPrice + value.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
	total.Add(total, tx.Value())
	return total
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.inner.rawSignatureValues()
}

// GasFeeCapCmp compares the fee cap of two transactions.
func (tx *Transaction) GasFeeCapCmp(other *Transaction) int {
	return tx.inner.gasFeeCap().Cmp(other.inner.gasFeeCap())
}

// GasFeeCapIntCmp compares the fee cap of the transaction against the given fee cap.
func (tx *Transaction) GasFeeCapIntCmp(other *big.Int) int {
	return tx.inner.gasFeeCap().Cmp(other)
}

// GasTipCapCmp compares the gasTipCap of two transactions.
func (tx *Transaction) GasTipCapCmp(other *Transaction) int {
	return tx.inner.gasTipCap().Cmp(other.inner.gasTipCap())
}

// GasTipCapIntCmp compares the gasTipCap of the transaction against the given gasTipCap.
func (tx *Transaction) GasTipCapIntCmp(other *big.Int) int {
	return tx.inner.gasTipCap().Cmp(other)
}

// Hash returns the transaction hash.
func (tx *Transaction) Hash() Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(Hash)
	}

	var h Hash
	if tx.Type() == LegacyTxType {
		h = rlpHash([]interface{}{
			tx.Nonce(),
			tx.GasPrice(),
			tx.Gas(),
			tx.To(),
			tx.Value(),
			tx.Data(),
			3, uint(0), uint(0),
		})
	} else {
		h = prefixedRlpHash(tx.Type(),
			[]interface{}{
				3,
				tx.Nonce(),
				tx.GasPrice(),
				tx.Gas(),
				tx.To(),
				tx.Value(),
				tx.Data(),
				tx.AccessList(),
			})
	}
	tx.hash.Store(h)
	return h
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previously cached value.
func (tx *Transaction) Size() StorageSize {
	return 0
}

// copyAddressPtr copies an address.
func copyAddressPtr(a *Address) *Address {
	if a == nil {
		return nil
	}
	cpy := *a
	return &cpy
}
