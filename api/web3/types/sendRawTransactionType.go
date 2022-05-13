package types

import "C"
import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	types "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	secp "github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
)

const (
	createContract = iota
	transfer
	invokeContract
	unknown
)

type SendRawTransactionRequestType struct {
	RawTransaction types.Bytes
}

type SendRawTransactionResponseType struct {
}

//unmarshaler rawTransaction
func ConstructTopiaTransaction(rawTransaction SendRawTransactionRequestType) (*txbasic.Transaction, error) {
	tx := new(Transaction)
	err := tx.UnmarshalBinary(rawTransaction.RawTransaction)
	if err != nil {
		return nil, errors.New("unmarshal rawTransaction error")
	}

	V, R, S := tx.RawSignatureValues()
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	v := ComputeV(V, tx.Type(), tx.ChainId())
	if v.BitLen() == 0 {
		sig[64] = uint8(0)
	} else {
		sig[64] = v.Bytes()[0]
	}

	var c secp.CryptServiceSecp256
	sigHash := tx.Hash().Bytes()
	pubKey, err := c.RecoverPublicKey(sigHash, sig)
	if err != nil {
		return nil, errors.New("recover PubKey error")
	}

	var from Address
	copy(from[:], Keccak256(pubKey[1:])[12:])
	fmt.Println(hex.EncodeToString(from.Bytes()))

	var txType int
	if tx.To() == nil && tx.Data() != nil {
		txType = createContract
	} else if tx.To() != nil && len(tx.Data()) == 0 {
		txType = transfer
	} else if tx.To() != nil && len(tx.Data()) != 0 {
		txType = invokeContract
	} else {
		txType = unknown
	}

	switch txType {
	case createContract:
		transactionUniversalHead := universal.TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          from.Bytes(),
			GasPrice:          tx.GasPrice().Uint64(),
			GasLimit:          tx.Gas(),
			Type:              uint32(universal.TransactionUniversalType_ContractDeploy),
			FeePayerSignature: []byte(rawTransaction.RawTransaction),
		}
		txData, _ := json.Marshal(tx.Data())
		txUni := universal.TransactionUniversal{
			Head: &transactionUniversalHead,
			Data: &universal.TransactionUniversalData{
				Specification: txData,
			},
		}

		transactionHead := txbasic.TransactionHead{
			Category:  []byte(txbasic.TransactionCategory_Eth),
			ChainID:   null,
			Version:   uint32(txbasic.Transaction_Eth_V1),
			FromAddr:  from.Bytes(),
			Nonce:     tx.Nonce(),
			Signature: []byte(rawTransaction.RawTransaction),
		}
		txDataBytes, _ := json.Marshal(txUni)
		tx := txbasic.Transaction{
			Head: &transactionHead,
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case transfer:
		transactionHead := txbasic.TransactionHead{
			Category: []byte(txbasic.TransactionCategory_Eth),
			ChainID:  null,
			Version:  uint32(txbasic.Transaction_Eth_V1),
			FromAddr: from.Bytes(),
			Nonce:    tx.Nonce(),
		}
		transactionUniversalHead := universal.TransactionUniversalHead{
			Version:  txbasic.Transaction_Topia_Universal_V1,
			FeePayer: from.Bytes(),
			GasPrice: tx.GasPrice().Uint64(),
			GasLimit: tx.Gas(),
			Type:     uint32(universal.TransactionUniversalType_Transfer),
		}
		targets := []universal.TargetItem{
			{
				currency.TokenSymbol_Native,
				tx.Value(),
			},
		}
		txfer := universal.TransactionUniversalTransfer{
			TransactionHead:          transactionHead,
			TransactionUniversalHead: transactionUniversalHead,
			TargetAddr:               tpcrtypes.Address(hex.EncodeToString(tx.To().Bytes())),
			Targets:                  targets,
		}

		txferData, _ := json.Marshal(txfer)

		txUni := universal.TransactionUniversal{
			Head: &txfer.TransactionUniversalHead,
			Data: &universal.TransactionUniversalData{
				Specification: txferData,
			},
		}
		txDataBytes, _ := json.Marshal(txUni)
		tx := txbasic.Transaction{
			Head: &transactionHead,
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case invokeContract:
		transactionUniversalHead := universal.TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          from.Bytes(),
			GasPrice:          tx.GasPrice().Uint64(),
			GasLimit:          tx.Gas(),
			Type:              uint32(universal.TransactionUniversalType_ContractInvoke),
			FeePayerSignature: []byte(rawTransaction.RawTransaction),
		}
		txData, _ := json.Marshal(tx.Data())
		txUni := universal.TransactionUniversal{
			Head: &transactionUniversalHead,
			Data: &universal.TransactionUniversalData{
				Specification: txData,
			},
		}

		transactionHead := txbasic.TransactionHead{
			Category: []byte(txbasic.TransactionCategory_Eth),
			ChainID:  null,
			Version:  uint32(txbasic.Transaction_Eth_V1),
			FromAddr: from.Bytes(),
			Nonce:    tx.Nonce(),
		}
		txDataBytes, _ := json.Marshal(txUni)
		tx := txbasic.Transaction{
			Head: &transactionHead,
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case unknown:
	default:
		return nil, nil
	}
	return nil, nil
}

func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}

func ComputeV(v *big.Int, txType byte, chainId *big.Int) *big.Int {
	switch txType {
	case LegacyTxType:
		if chainId.BitLen() == 0 {
			v = new(big.Int).Sub(v, big.NewInt(27))
		} else {
			v = new(big.Int).Sub(v, big.NewInt(35))
			v = new(big.Int).Sub(v, new(big.Int).Mul(big.NewInt(2), chainId))
		}
	case AccessListTxType:
	case DynamicFeeTxType:
	}
	return v
}
