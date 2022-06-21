package handlers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	types2 "github.com/TopiaNetwork/topia/api/web3/eth/types"
	"github.com/TopiaNetwork/topia/api/web3/eth/types/eth_account"
	"github.com/TopiaNetwork/topia/api/web3/eth/types/eth_transaction"
	hexutil "github.com/TopiaNetwork/topia/api/web3/eth/types/hexutil"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	secp "github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
	"strconv"
)

const (
	createContract = iota
	transfer
	invokeContract
	unknown
	SignatureLength = 64 + 1
)

func ConvertEthTransactionToTopiaTransaction(tx *eth_transaction.Transaction) (*txbasic.Transaction, error) {
	var v, r, s *big.Int
	v, r, s = tx.RawSignatureValues()

	sig := make([]byte, SignatureLength)
	copy(sig[32-len(r.Bytes()):32], r.Bytes())
	copy(sig[64-len(s.Bytes()):64], s.Bytes())
	v = ComputeV(v, tx.Type(), tx.ChainId())
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
	var addr eth_account.Address
	copy(addr[:], eth_account.Keccak256(pubKey[1:])[12:])
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

	txData, _ := marshaler.Marshal(tx.Data())
	txUni := txuni.TransactionUniversal{
		Head: &txuni.TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          from.Bytes(),
			GasPrice:          tx.GasPrice().Uint64(),
			GasLimit:          tx.Gas(),
			FeePayerSignature: sig,
		},
		Data: &txuni.TransactionUniversalData{
			Specification: txData,
		},
	}
	txDataBytes, _ := marshaler.Marshal(txUni)
	txHead := &txbasic.TransactionHead{
		Category:  []byte(txbasic.TransactionCategory_Topia_Universal),
		ChainID:   tx.ChainId().Bytes(),
		Version:   uint32(txbasic.Transaction_Topia_Universal_V1),
		FromAddr:  from.Bytes(),
		Nonce:     tx.Nonce(),
		Signature: sig,
	}

	switch txFunc {
	case createContract:
		txUni.GetHead().Type = uint32(txuni.TransactionUniversalType_ContractDeploy)
		tx := createTransaction(txHead, txDataBytes)
		return tx, nil
	case transfer:
		txUni.GetHead().Type = uint32(txuni.TransactionUniversalType_Transfer)
		txfer := txuni.TransactionUniversalTransfer{
			TransactionHead:          *txHead,
			TransactionUniversalHead: *txUni.GetHead(),
			TargetAddr:               tpcrtypes.Address(hex.EncodeToString(tx.To().Bytes())),
			Targets: []txuni.TargetItem{
				{
					currency.TokenSymbol_Native,
					tx.Value(),
				},
			},
		}
		txferData, _ := json.Marshal(txfer)
		txUni.GetData().Specification = txferData
		txDataBytes, _ = json.Marshal(txUni)
		tx := createTransaction(txHead, txDataBytes)
		return tx, nil
	case invokeContract:
		txUni.GetHead().Type = uint32(txuni.TransactionUniversalType_ContractInvoke)
		tx := createTransaction(txHead, txDataBytes)
		return tx, nil
	case unknown:
		return nil, errors.New("unknown tx func,only support transfer/createContract/InvokeContract!")
	}
	return nil, errors.New("convert ethTransactionTotopiaTransaction failed!")
}

func createTransaction(head *txbasic.TransactionHead, specification []byte) *txbasic.Transaction {
	return &txbasic.Transaction{
		Head: head,
		Data: &txbasic.TransactionData{
			Specification: specification,
		},
	}
}

func ConvertTopiaBlockToEthBlock(block *tpchaintypes.Block) *GetBlockResponseType {
	transactionHashs := block.GetData().GetTxs()
	transactionHashString := make([]interface{}, 0)
	for _, v := range transactionHashs {
		var hash eth_account.Hash
		hash.SetBytes(v)
		transactionHashString = append(transactionHashString, hash.Hex())
	}

	gas := big.NewInt(10_000_000_000)
	height := new(big.Int).SetUint64(block.GetHead().GetHeight())
	baseFee := big.NewInt(0)
	var blockHash eth_account.Hash
	blockHash.SetBytes(block.GetHead().GetHash())
	var parentBlockHash eth_account.Hash
	parentBlockHash.SetBytes(block.GetHead().GetParentBlockHash())
	var stateRoot eth_account.Hash
	stateRoot.SetBytes(block.GetHead().GetStateRoot())
	var miner eth_account.Address
	miner.SetBytes(block.GetHead().GetLauncher())
	var gasUsed hexutil.Uint64
	json.Unmarshal(block.GetHead().GetGasFees(), &gasUsed)
	var txToor eth_account.Hash
	txToor.SetBytes(block.GetHead().GetTxRoot())
	var txReceiptRoot eth_account.Hash
	txReceiptRoot.SetBytes(block.GetHead().GetTxResultRoot())
	result := &GetBlockResponseType{
		Number:           (*hexutil.Big)(height),
		BaseFeePerGas:    (*hexutil.Big)(baseFee),
		Hash:             blockHash,
		ParentHash:       parentBlockHash,
		Nonce:            []byte{},
		StateRoot:        stateRoot,
		Miner:            miner,
		Size:             hexutil.Uint64(block.Size()),
		GasLimit:         hexutil.Uint64(gas.Uint64()),
		GasUsed:          gasUsed,
		Timestamp:        hexutil.Uint64(block.GetHead().GetTimeStamp()),
		TransactionsRoot: txToor,
		ReceiptsRoot:     txReceiptRoot,
		Transactions:     transactionHashString,
	}
	return result
}

func ConvertTopiaResultToEthReceipt(transactionResult txbasic.TransactionResult, apiServant servant.APIServant) map[string]interface{} {
	var resultData txuni.TransactionResultUniversal
	var txHash []byte
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	if txbasic.TransactionCategory(transactionResult.GetHead().GetCategory()) == txbasic.TransactionCategory_Topia_Universal {
		_ = marshaler.Unmarshal(transactionResult.GetData().GetSpecification(), &resultData)
		txHash = resultData.GetTxHash()
	}
	servant := apiServant
	transaction, _ := servant.GetTransactionByHash(string(txHash))
	block, _ := servant.GetBlockByTxHash(string(txHash))
	var transactionUniversal txuni.TransactionUniversal
	json.Unmarshal(transaction.GetData().GetSpecification(), &transactionUniversal)
	var transactionUniversalTransfer txuni.TransactionUniversalTransfer
	json.Unmarshal(transactionUniversal.GetData().GetSpecification(), &transactionUniversalTransfer)

	status, _ := strconv.ParseInt(resultData.GetStatus().String(), 10, 64)
	var sta *big.Int
	if status == 1 {
		sta = big.NewInt(0)
	} else {
		sta = big.NewInt(1)
	}

	bloom, _ := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000")

	var blockH eth_account.Hash
	blockH.SetBytes(block.GetHead().GetHash())
	var tranH eth_account.Hash
	tranH.SetBytes(txHash)
	var from eth_account.Address
	from.SetBytes(transaction.GetHead().GetFromAddr())
	var conAddr eth_account.Address
	conAddr.SetBytes(resultData.GetData())
	result := map[string]interface{}{
		"blockHash":         blockH,
		"blockNumber":       hexutil.Uint64(block.GetHead().GetHeight()),
		"transactionHash":   tranH,
		"transactionIndex":  hexutil.Uint64(uint64(1)),
		"from":              from,
		"to":                nil,
		"gasUsed":           hexutil.Uint64(resultData.GetGasUsed()),
		"cumulativeGasUsed": hexutil.Uint64(resultData.GetGasUsed()),
		"contractAddress":   conAddr,
		"logs":              []*eth_account.Address{},
		"logsBloom":         types2.BytesToBloom(bloom),
		"type":              hexutil.Uint(2),
		"effectiveGasPrice": hexutil.Uint64(0),
		"status":            hexutil.Uint(sta.Uint64()),
	}
	return result
}

func TopiaTransactionToEthTransaction(tx txbasic.Transaction, apiServant servant.APIServant) (*EthRpcTransaction, error) {
	v, r, s := RawSignatureValues(tx)
	chainId := tx.GetHead().GetChainID()
	chainIdBigInt := new(big.Int).SetBytes(chainId)
	bh, blockNumber := blockInfo(tx, apiServant)
	gasPrice := new(big.Int).SetUint64(GasPrice(&tx))
	index := tx.GetHead().GetIndex()
	height := new(big.Int).SetUint64(blockNumber)

	var from eth_account.Address
	from.SetBytes(tx.Head.FromAddr)
	var to eth_account.Address
	to.SetBytes(ToAddress(&tx).Bytes())
	var hash eth_account.Hash
	hashByte, _ := tx.HashBytes()
	hash.SetBytes(hashByte)
	var blockHash eth_account.Hash
	blockHash.SetBytes(bh)
	result := &EthRpcTransaction{
		ChainID:          (*hexutil.Big)(chainIdBigInt),
		Type:             hexutil.Uint64(0),
		From:             from,
		Gas:              hexutil.Uint64(GasLimit(&tx)),
		GasPrice:         (*hexutil.Big)(gasPrice),
		Hash:             hash,
		Input:            hexutil.Bytes{},
		Nonce:            hexutil.Uint64(Nonce(&tx)),
		To:               to,
		Value:            (*hexutil.Big)(Value(&tx)),
		TransactionIndex: (*hexutil.Uint64)(&index),
		BlockHash:        blockHash,
		BlockNumber:      (*hexutil.Big)(height),
		V:                (*hexutil.Big)(v),
		R:                (*hexutil.Big)(r),
		S:                (*hexutil.Big)(s),
	}
	return result, nil
}

func AddPrefix(s string) string {
	return "0x" + s
}

func ToAddress(tx *txbasic.Transaction) tpcrtypes.Address {
	var toAddress tpcrtypes.Address
	var targetData txuni.TransactionUniversalTransfer
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Topia_Universal {
		var txData txuni.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txData)
		if txData.Head.Type == uint32(txuni.TransactionUniversalType_Transfer) {
			_ = json.Unmarshal(txData.Data.Specification, &targetData)
			toAddress = targetData.TargetAddr
		}
	} else {
		return ""
	}
	return toAddress
}

func GasPrice(tx *txbasic.Transaction) uint64 {
	var gasPrice uint64
	switch txbasic.TransactionCategory(tx.Head.Category) {
	case txbasic.TransactionCategory_Topia_Universal:
		var txUniver txuni.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txUniver)
		gasPrice = txUniver.Head.GasPrice
		return gasPrice
	default:
		return 0
	}
}

func GasLimit(tx *txbasic.Transaction) uint64 {
	var gasLimit uint64
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Topia_Universal {
		var txData txuni.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txData)
		gasLimit = txData.Head.GasLimit
		return gasLimit
	} else {
		return 0
	}
}

func Nonce(tx *txbasic.Transaction) uint64 {
	var nonce uint64
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Topia_Universal {
		nonce = tx.GetHead().GetNonce()
		return nonce
	} else {
		return 0
	}
}

func Value(tx *txbasic.Transaction) *big.Int {
	var value *big.Int
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Topia_Universal {
		var txData txuni.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txData)
		var txDataSpe txuni.TransactionUniversalTransfer
		_ = json.Unmarshal(txData.GetData().Specification, &txDataSpe)
		value = txDataSpe.Targets[0].Value
		return value
	} else {
		return big.NewInt(0)
	}
}

func RawSignatureValues(tx txbasic.Transaction) (v, r, s *big.Int) {
	var txData txuni.TransactionUniversal
	_ = json.Unmarshal(tx.Data.Specification, &txData)

	sig := tx.GetHead().GetSignature()
	r, s, v = decodeSignature(sig)

	chainId := tx.GetHead().GetChainID()
	chainIdBigInt := new(big.Int).SetBytes(chainId)
	if chainIdBigInt.Sign() != 0 {
		v = big.NewInt(int64(sig[64] + 35))
		v.Add(v, big.NewInt(2))
	}
	return v, r, s
}

func decodeSignature(sig []byte) (r, s, v *big.Int) {

	if len(sig) != SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), SignatureLength))
	}

	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}

func blockInfo(tx txbasic.Transaction, apiServant servant.APIServant) (blockHash []byte, blockNumber uint64) {

	servant := apiServant
	txHash, _ := tx.HashHex()
	block, _ := servant.GetBlockByTxHash(txHash)

	blockHash, _ = block.HashBytes()

	blockNumber = block.GetHead().GetHeight()
	return blockHash, blockNumber
}
