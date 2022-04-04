package universal

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/codec"

	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func ConstructTransactionWithUniversalTransfer(
	log tplog.Logger,
	cryptService tpcrt.CryptService,
	fromPriKey tpcrtypes.PrivateKey,
	feePayerPriKey tpcrtypes.PrivateKey,
	nonce uint64,
	gasPrice uint64,
	gasLimit uint64,
	targetAddr tpcrtypes.Address,
	targets []TargetItem) *txbasic.Transaction {

	trDataBytes, err := json.Marshal(&struct {
		TargetAddr tpcrtypes.Address
		Targets    []TargetItem
	}{
		targetAddr,
		targets,
	})
	if err != nil {
		panic("Invalid transfer data info: " + err.Error())
	}

	txUniSignData, err := cryptService.Sign(feePayerPriKey, trDataBytes)
	if err != nil {
		panic("Sign transfer data err: " + err.Error())
	}
	feePayerPubKey, err := cryptService.ConvertToPublic(feePayerPriKey)
	if err != nil {
		panic("Can't convert public key from feePayerPriKey: " + err.Error())
	}
	feePayerAddr, err := cryptService.CreateAddress(feePayerPubKey)
	if err != nil {
		panic("Can't convert public key from feePayerPriKey: " + err.Error())
	}

	txUniSignBytes, err := json.Marshal(&tpcrtypes.SignatureInfo{
		SignData:  txUniSignData,
		PublicKey: feePayerPubKey,
	})
	if err != nil {
		panic("Marshal fee payer signature info err: " + err.Error())
	}

	txUni := &TransactionUniversal{
		Head: &TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          []byte(feePayerAddr),
			Nonce:             nonce,
			GasPrice:          gasPrice,
			GasLimit:          gasLimit,
			Type:              uint32(TransactionUniversalType_Transfer),
			FeePayerSignature: txUniSignBytes,
		},
		Data: &TransactionUniversalData{Specification: trDataBytes},
	}

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	txDataBytes, err := marshaler.Marshal(txUni)
	if err != nil {
		panic("Marshal tx universal err: " + err.Error())
	}

	return txbasic.NewTransaction(log, cryptService, fromPriKey, txbasic.TransactionCategory_Topia_Universal, txbasic.Transaction_Topia_Universal_V1, txDataBytes)
}
