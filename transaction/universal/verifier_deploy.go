package universal

import (
	"context"

	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionUniversalDeployVerifier func(ctx context.Context, log tplog.Logger, txDP *TransactionUniversalDeploy, txUniServant TransactionUniversalServant) txbasic.VerifyResult

func TransactionUniversalDeployContractAddressVerifier() TransactionUniversalDeployVerifier {
	return func(ctx context.Context, log tplog.Logger, txDP *TransactionUniversalDeploy, txServant TransactionUniversalServant) txbasic.VerifyResult {
		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalDeployCodeVerifier() TransactionUniversalDeployVerifier {
	return func(ctx context.Context, log tplog.Logger, txDP *TransactionUniversalDeploy, txServant TransactionUniversalServant) txbasic.VerifyResult {
		codeSize := uint64(len(txDP.Code))
		if codeSize > txServant.GetChainConfig().MaxCodeSize {
			log.Errorf("Deploy code size %d reaches max size", codeSize, txServant.GetChainConfig().MaxCodeSize)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func ApplyTransactionUniversalDeployVerifiers(ctx context.Context, log tplog.Logger, txDP *TransactionUniversalDeploy, txUniServant TransactionUniversalServant, verifiers ...TransactionUniversalDeployVerifier) txbasic.VerifyResult {
	vrResult := txbasic.VerifyResult_Accept
	for _, verifier := range verifiers {
		vR := verifier(ctx, log, txDP, txUniServant)
		switch vR {
		case txbasic.VerifyResult_Reject:
			return txbasic.VerifyResult_Reject
		case txbasic.VerifyResult_Ignore:
			vrResult = vR
		}
	}

	return vrResult
}
