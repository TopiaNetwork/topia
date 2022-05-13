package convert

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/api/web3/types"
	"github.com/TopiaNetwork/topia/codec"
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
	SignatureLength = 64 + 1
)

func ConvertEthTransactionToTopiaTransaction(tx *types.Transaction) (*txbasic.Transaction, error) {
	var v, r, s *big.Int
	v, r, s = tx.RawSignatureValues()

	sig := make([]byte, SignatureLength)
	copy(sig[32-len(r.Bytes()):32], r.Bytes())
	copy(sig[64-len(s.Bytes()):64], s.Bytes())
	v = types.ComputeV(v, tx.Type(), tx.ChainId())
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
	var addr types.Address
	copy(addr[:], types.Keccak256(pubKey[1:])[12:])
	from := tpcrtypes.NewFromBytes(addr.Bytes())

	var txFunc int
	if tx.To() == nil && tx.Data() != nil {
		txFunc = createContract
	} else if tx.To() != nil && tx.Data() == nil {
		txFunc = transfer
	} else if tx.To() != nil && tx.Data() != nil {
		txFunc = invokeContract
	} else {
		txFunc = unknown
	}
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	switch txFunc {
	case createContract:
		txData, _ := marshaler.Marshal(tx.Data())

		txUni := universal.TransactionUniversal{
			Head: &universal.TransactionUniversalHead{
				Version:           txbasic.Transaction_Topia_Universal_V1,
				FeePayer:          from.Bytes(),
				GasPrice:          tx.GasPrice().Uint64(),
				GasLimit:          tx.Gas(),
				Type:              uint32(universal.TransactionUniversalType_ContractDeploy),
				FeePayerSignature: sig,
			},
			Data: &universal.TransactionUniversalData{
				Specification: txData,
			},
		}
		txDataBytes, _ := marshaler.Marshal(txUni)

		tx := txbasic.Transaction{
			Head: &txbasic.TransactionHead{
				Category:  []byte(txbasic.TransactionCategory_Eth),
				ChainID:   tx.ChainId().Bytes(),
				Version:   uint32(txbasic.Transaction_Eth_V1),
				FromAddr:  from.Bytes(),
				Nonce:     tx.Nonce(),
				Signature: sig,
			},
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case transfer:
		transactionHead := txbasic.TransactionHead{
			Category:  []byte(txbasic.TransactionCategory_Eth),
			ChainID:   tx.ChainId().Bytes(),
			Version:   uint32(txbasic.Transaction_Eth_V1),
			FromAddr:  from.Bytes(),
			Nonce:     tx.Nonce(),
			Signature: sig,
		}
		transactionUniversalHead := universal.TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          from.Bytes(),
			GasPrice:          tx.GasPrice().Uint64(),
			GasLimit:          tx.Gas(),
			Type:              uint32(universal.TransactionUniversalType_Transfer),
			FeePayerSignature: sig,
		}

		txfer := universal.TransactionUniversalTransfer{
			TransactionHead:          transactionHead,
			TransactionUniversalHead: transactionUniversalHead,
			TargetAddr:               tpcrtypes.Address(hex.EncodeToString(tx.To().Bytes())),
			Targets: []universal.TargetItem{
				{
					currency.TokenSymbol_Native,
					tx.Value(),
				},
			},
		}
		txferData, _ := json.Marshal(txfer)

		txUni := universal.TransactionUniversal{
			Head: &transactionUniversalHead,
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
		txData, _ := marshaler.Marshal(tx.Data())

		txUni := universal.TransactionUniversal{
			Head: &universal.TransactionUniversalHead{
				Version:           txbasic.Transaction_Topia_Universal_V1,
				FeePayer:          from.Bytes(),
				GasPrice:          tx.GasPrice().Uint64(),
				GasLimit:          tx.Gas(),
				Type:              uint32(universal.TransactionUniversalType_ContractInvoke),
				FeePayerSignature: sig,
			},
			Data: &universal.TransactionUniversalData{
				Specification: txData,
			},
		}
		txDataBytes, _ := marshaler.Marshal(txUni)

		tx := txbasic.Transaction{
			Head: &txbasic.TransactionHead{
				Category:  []byte(txbasic.TransactionCategory_Eth),
				ChainID:   tx.ChainId().Bytes(),
				Version:   uint32(txbasic.Transaction_Eth_V1),
				FromAddr:  from.Bytes(),
				Nonce:     tx.Nonce(),
				Signature: sig,
			},
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case unknown:
		return nil, errors.New("unknown tx func,only support transfer/createContract/InvokeContract!")
	}
	return nil, errors.New("convert ethTransactionTotopiaTransaction failed!")
}
