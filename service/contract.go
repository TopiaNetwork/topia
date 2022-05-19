package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
)

type ContractService interface {
	Deploy(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, code []byte) (txbasic.TxID, tpcrtypes.Address, error)

	Invoke(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, ContractAddr tpcrtypes.Address, Method string, Args string) (txbasic.TxID, []byte, error)
}

type contractService struct {
	log               tplog.Logger
	marshaler         codec.Marshaler
	stateQueryService StateQueryService
	txService         TransactionService
	walletService     WalletService
}

func NewContractService(log tplog.Logger, marshaler codec.Marshaler, stateQueryService StateQueryService, txService TransactionService, walletService WalletService) ContractService {
	return &contractService{
		log:               log,
		marshaler:         marshaler,
		stateQueryService: stateQueryService,
		txService:         txService,
		walletService:     walletService,
	}
}

func (cs *contractService) getTxUniRS(txRS *txbasic.TransactionResult) (*txuni.TransactionResultUniversal, error) {
	if txbasic.TransactionCategory(txRS.Head.Category) != txbasic.TransactionCategory_Topia_Universal {
		return nil, fmt.Errorf("Invalid tx result: category=%s, expected TransactionCategory_Topia_Universal", txbasic.TransactionCategory(txRS.Head.Category))
	}

	var txUniRS txuni.TransactionResultUniversal
	err := cs.marshaler.Unmarshal(txRS.Data.Specification, &txUniRS)
	if err != nil {
		return nil, err
	}

	return &txUniRS, nil
}

func (cs *contractService) makeTransaction(fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, txUniType txuni.TransactionUniversalType, txUniDataBytes []byte) (*txbasic.Transaction, error) {
	if fromAddr == "" || fromAddr == tpcrtypes.UndefAddress {
		fromAddr, _ = cs.walletService.Default()
	}

	if payerAddr == "" || payerAddr == tpcrtypes.UndefAddress {
		payerAddr = fromAddr
	}

	payerSignInfo, err := cs.walletService.Sign(payerAddr, txUniDataBytes)
	if err != nil {
		return nil, err
	}
	payerSignInfoBytes, _ := json.Marshal(&payerSignInfo)
	txUniHead := &txuni.TransactionUniversalHead{
		Version:           uint32(txuni.TransactionUniversalVersion_v1),
		FeePayer:          []byte(payerAddr),
		GasPrice:          gasPrice,
		GasLimit:          gasLimit,
		Type:              uint32(txUniType),
		FeePayerSignature: payerSignInfoBytes,
	}
	txUniData := &txuni.TransactionUniversalData{Specification: txUniDataBytes}
	txUni := &txuni.TransactionUniversal{
		Head: txUniHead,
		Data: txUniData,
	}

	txDataBytes, _ := cs.marshaler.Marshal(txUni)
	txSignInfo, err := cs.walletService.Sign(fromAddr, txDataBytes)
	if err != nil {
		return nil, err
	}
	txSignInfoBytes, _ := json.Marshal(&txSignInfo)
	fromAcc, err := cs.stateQueryService.GetAccount(fromAddr)
	if err != nil {
		return nil, err
	}
	txHead := &txbasic.TransactionHead{
		Category:  []byte(txbasic.TransactionCategory_Topia_Universal),
		ChainID:   []byte(cs.stateQueryService.ChainID()),
		Version:   txbasic.Transaction_Topia_Universal_V1,
		TimeStamp: uint64(time.Now().UnixNano()),
		Nonce:     fromAcc.Nonce,
		Signature: txSignInfoBytes,
	}
	txData := &txbasic.TransactionData{Specification: txDataBytes}

	return &txbasic.Transaction{
		Head: txHead,
		Data: txData,
	}, nil
}

func (cs *contractService) doAction(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, txUniType txuni.TransactionUniversalType, txUniDataBytes []byte, actionName string) (txbasic.TxID, *txuni.TransactionResultUniversal, error) {
	tx, err := cs.makeTransaction(fromAddr, payerAddr, gasPrice, gasLimit, txUniType, txUniDataBytes)
	if err != nil {
		cs.log.Errorf("Make deployment transaction err: %v", err)
		return "", nil, err
	}

	txRS, err := cs.txService.ForwardTxSync(ctx, tx)
	if err != nil {
		cs.log.Errorf("Forward %s transaction err: %v", err, actionName)
		return "", nil, err
	}

	txUniRS, err := cs.getTxUniRS(txRS)
	if err != nil {
		cs.log.Errorf("Can't get tx uni result when %s: %v", err, actionName)
		return "", nil, err
	}

	txID, _ := tx.TxID()

	return txID, txUniRS, nil
}

func (cs *contractService) Deploy(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, code []byte) (txbasic.TxID, tpcrtypes.Address, error) {
	deployDataBytes, _ := json.Marshal(&struct {
		ContractAddress tpcrtypes.Address
		Code            []byte
	}{
		"",
		code,
	})

	txID, txUniRS, err := cs.doAction(ctx, fromAddr, payerAddr, gasPrice, gasLimit, txuni.TransactionUniversalType_ContractDeploy, deployDataBytes, "deployment")
	if err != nil {
		return "", tpcrtypes.UndefAddress, err
	}

	if txUniRS.Status != txuni.TransactionResultUniversal_OK {
		return txID, tpcrtypes.UndefAddress, fmt.Errorf("The final deployment result err: %s", string(txUniRS.ErrString))
	}

	return txID, tpcrtypes.NewFromBytes(txUniRS.Data), nil
}

func (cs *contractService) Invoke(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, ContractAddr tpcrtypes.Address, Method string, Args string) (txbasic.TxID, []byte, error) {
	invokeDataBytes, _ := json.Marshal(&struct {
		ContractAddress tpcrtypes.Address
		Method          string
		Args            string
	}{
		ContractAddr,
		Method,
		Args,
	})

	txID, txUniRS, err := cs.doAction(ctx, fromAddr, payerAddr, gasPrice, gasLimit, txuni.TransactionUniversalType_ContractInvoke, invokeDataBytes, "invoking")
	if err != nil {
		return "", nil, err
	}

	if txUniRS.Status != txuni.TransactionResultUniversal_OK {
		return txID, nil, fmt.Errorf("The final deployment result err: %s", string(txUniRS.ErrString))
	}

	return txID, txUniRS.Data, nil
}
