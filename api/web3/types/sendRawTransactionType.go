package types

import "C"
import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/api/web3/sha3"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

const (
	createContract = iota
	transfer
	invokeContract
	unknown
	SignatureLength = 64 + 1
)

type SendRawTransactionRequestType struct {
	RawTransaction string
}

type SendRawTransactionResponseType struct {
}

//解析rawTransaction
func ConstructTopiaTransaction(rawTransaction SendRawTransactionRequestType) (*txbasic.Transaction, error) {
	//1.反序列化rawTransaction
	tx := new(types.Transaction)
	txbyte, err := hex.DecodeString(rawTransaction.RawTransaction[2:])
	if err != nil {
		fmt.Println(err)
	}
	tx.UnmarshalBinary(txbyte)

	//检查这里反序列化的对不对
	//解析正确

	signer := types.NewLondonSigner(big.NewInt(3))
	from, err := types.Sender(signer, tx)
	//
	//fmt.Println(hex.EncodeToString(from[:]))
	//
	//return nil, nil
	//
	//2.计算from地址
	//V, R, S := tx.RawSignatureValues()
	////根据交易类型更新v
	//V = computeV(V, tx.inner.txType())
	//V = big.NewInt(29)
	////这里的哈希计算错误，原因是前缀设置的不对
	//from, err := recoverPlain(tx.Hash(), R, S, V, true)
	//ff := hex.EncodeToString(from[:])
	//fmt.Println(ff)
	//if err != nil {
	//	return nil, err
	//}

	//3.根据ethtx交易类型，构建对应的topiaTransaction
	//3.1获取ethtx交易类型
	var txType int
	if tx.To() == nil && tx.Data() != nil {
		txType = createContract
	} else if tx.To() != nil && tx.Data() == nil {
		txType = transfer
	} else if tx.To() != nil && tx.Data() != nil {
		txType = invokeContract
	} else {
		txType = unknown
	}
	//构造topia交易
	switch txType {
	case createContract:
		transactionUniversalHead := universal.TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          from[:],
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
			Category: []byte(txbasic.TransactionCategory_Eth),
			ChainID:  null,
			Version:  uint32(txbasic.Transaction_Eth_V1),
			FromAddr: from[:],
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
	case transfer:
		transactionHead := txbasic.TransactionHead{
			Category: []byte(txbasic.TransactionCategory_Eth),
			ChainID:  null,
			Version:  uint32(txbasic.Transaction_Eth_V1),
			FromAddr: from[:],
			Nonce:    tx.Nonce(),
		}
		transactionUniversalHead := universal.TransactionUniversalHead{
			Version:  txbasic.Transaction_Topia_Universal_V1,
			FeePayer: from[:],                //from地址
			GasPrice: tx.GasPrice().Uint64(), //gasPrice
			GasLimit: tx.Gas(),               //gasLimit
			Type:     uint32(universal.TransactionUniversalType_Transfer),
		}
		targets := []universal.TargetItem{
			{
				currency.TokenSymbol_Native,
				tx.Value(), //转账金额
			},
		}
		txfer := universal.TransactionUniversalTransfer{
			TransactionHead:          transactionHead,
			TransactionUniversalHead: transactionUniversalHead,
			TargetAddr:               tpcrtypes.Address(tx.To()[:]), //目标地址
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
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case invokeContract:
		transactionUniversalHead := universal.TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          from[:],
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
			FromAddr: from[:],
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

func rlpHash(x interface{}) (h Hash) {
	//marshaler := codec.CreateMarshaler(codec.CodecType_RLP)
	//data, err := marshaler.Marshal(x)
	//if err != nil {
	//	fmt.Println(data)
	//}
	data := x.([]byte)
	hash := sha3.SHA3(
		data,
	)
	copy(h[:], hash[:])
	return h
}

func prefixedRlpHash(prefix byte, x interface{}) (h Hash) {
	//此处需要拼接prefix
	marshaler := codec.CreateMarshaler(codec.CodecType_RLP)
	data, err := marshaler.Marshal(x)
	if err != nil {
		fmt.Println(data)
	}
	da := make([]byte, len(data))
	da[0] = prefix
	copy(da[1:], data[:])
	//sha3.SoliditySHA3WithPrefix(data)
	hash := sha3.SHA3(
		da,
	)
	copy(h[:], hash[:])
	return h
}

func recoverPlain(sighash Hash, R, S, Vb *big.Int, homestead bool) (Address, error) {
	if Vb.BitLen() > 8 {
		return Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature//TODO：没实现
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return Address{}, err
	}
	var addr Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

func computeV(v *big.Int, txType byte) *big.Int {
	switch txType {
	case LegacyTxType:
		v = new(big.Int).Sub(v, big.NewInt(2))
		v.Sub(v, big.NewInt(8))
	case AccessListTxType:
	case DynamicFeeTxType:
		v = new(big.Int).Add(v, big.NewInt(27))
	}
	return v
}
