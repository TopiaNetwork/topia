package universal

import (
	"context"

	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionUniversalTransferVerifier func(ctx context.Context, log tplog.Logger, txTr *TransactionUniversalTransfer, txUniServant TransactionUniversalServant) txbasic.VerifyResult

func TransactionUniversalTransferTargetAddressVerifier() TransactionUniversalTransferVerifier {
	return func(ctx context.Context, log tplog.Logger, txTr *TransactionUniversalTransfer, txServant TransactionUniversalServant) txbasic.VerifyResult {
		targetAddr := txTr.TargetAddr

		cryType, err := targetAddr.CryptType()
		if err != nil {
			log.Errorf("Can't get from address type: %v", err)
			return txbasic.VerifyResult_Reject
		}

		if isValid, _ := targetAddr.IsValid(tpnet.CurrentNetworkType, cryType); !isValid {
			log.Errorf("Invalid target address: %v", txTr.TargetAddr)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalTransferTargetItemsVerifier() TransactionUniversalTransferVerifier {
	return func(ctx context.Context, log tplog.Logger, txTr *TransactionUniversalTransfer, txServant TransactionUniversalServant) txbasic.VerifyResult {
		targetItemSize := uint64(len(txTr.Targets))
		if targetItemSize > txServant.GetChainConfig().MaxTargetItem {
			log.Errorf("Transfer target item size %d reaches max size", targetItemSize, txServant.GetChainConfig().MaxTargetItem)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func ApplyTransactionUniversalTransferVerifiers(ctx context.Context, log tplog.Logger, txTr *TransactionUniversalTransfer, txUniServant TransactionUniversalServant, verifiers ...TransactionUniversalTransferVerifier) txbasic.VerifyResult {
	vrResult := txbasic.VerifyResult_Accept
	for _, verifier := range verifiers {
		vR := verifier(ctx, log, txTr, txUniServant)
		switch vR {
		case txbasic.VerifyResult_Reject:
			return txbasic.VerifyResult_Reject
		case txbasic.VerifyResult_Ignore:
			vrResult = vR
		}
	}

	return vrResult
}
