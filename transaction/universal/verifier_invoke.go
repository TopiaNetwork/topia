package universal

import (
	"context"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"regexp"
)

type TransactionUniversalInvokeVerifier func(ctx context.Context, log tplog.Logger, txIV *TransactionUniversalInvoke, txUniServant TransactionUniversalServant) txbasic.VerifyResult

func TransactionUniversalInvokeContractAddressVerifier() TransactionUniversalInvokeVerifier {
	return func(ctx context.Context, log tplog.Logger, txIV *TransactionUniversalInvoke, txServant TransactionUniversalServant) txbasic.VerifyResult {
		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalInvokeMethodVerifier() TransactionUniversalInvokeVerifier {
	return func(ctx context.Context, log tplog.Logger, txIV *TransactionUniversalInvoke, txServant TransactionUniversalServant) txbasic.VerifyResult {
		if len(txIV.Method) == 0 {
			log.Error("There is no invoke method")
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalInvokeArgsVerifier() TransactionUniversalInvokeVerifier {
	return func(ctx context.Context, log tplog.Logger, txIV *TransactionUniversalInvoke, txServant TransactionUniversalServant) txbasic.VerifyResult {
		re, err := regexp.Compile(`^[a-zA-Z0-9@]`)
		if err != nil {
			log.Error("There is no invoke method")
			return txbasic.VerifyResult_Reject
		}
		match := re.MatchString(txIV.Args)
		if !match {
			log.Error("There is no invoke method")
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func ApplyTransactionUniversalInvokeVerifiers(ctx context.Context, log tplog.Logger, txIV *TransactionUniversalInvoke, txUniServant TransactionUniversalServant, verifiers ...TransactionUniversalInvokeVerifier) txbasic.VerifyResult {
	vrResult := txbasic.VerifyResult_Accept
	for _, verifier := range verifiers {
		vR := verifier(ctx, log, txIV, txUniServant)
		switch vR {
		case txbasic.VerifyResult_Reject:
			return txbasic.VerifyResult_Reject
		case txbasic.VerifyResult_Ignore:
			vrResult = vR
		}
	}

	return vrResult
}
