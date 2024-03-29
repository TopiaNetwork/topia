package universal

import (
	"context"
	"encoding/json"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionUniversalVerifier func(ctx context.Context, log tplog.Logger, nodeID string, txUni *TransactionUniversalWithHead, txUniServant TransactionUniversalServant) txbasic.VerifyResult

func TransactionUniversalPayerAddressVerifier() TransactionUniversalVerifier {
	return func(ctx context.Context, log tplog.Logger, nodeID string, txUni *TransactionUniversalWithHead, txUniServant TransactionUniversalServant) txbasic.VerifyResult {
		payerAddr := tpcrtypes.NewFromBytes(txUni.Head.FeePayer)

		if isValid := payerAddr.IsValid(tpcmm.CurrentNetworkType); !isValid {
			log.Errorf("Invalid payer address: %v", txUni.Head.FeePayer)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalGasVerifier() TransactionUniversalVerifier {
	return func(ctx context.Context, log tplog.Logger, nodeID string, txUni *TransactionUniversalWithHead, txUniServant TransactionUniversalServant) txbasic.VerifyResult {
		if txUni.Head.GasPrice < txUniServant.GetGasConfig().MinGasPrice {
			log.Errorf("GasPrice %d less than the min gas price %d", txUni.Head.GasPrice, txUniServant.GetGasConfig().MinGasPrice)
			return txbasic.VerifyResult_Reject
		}

		if txUni.Head.GasLimit < txUniServant.GetGasConfig().MinGasLimit {
			log.Errorf("GasLimit %d less than the min gas limit %d", txUni.Head.GasLimit, txUniServant.GetGasConfig().MinGasLimit)
			return txbasic.VerifyResult_Reject
		}

		gasEstimator, err := txUniServant.GetGasEstimator()
		if err != nil {
			log.Errorf("Can't get gas estimator: %v", err)
			return txbasic.VerifyResult_Reject
		}
		estiGas, err := gasEstimator.Estimate(ctx, log, nodeID, txUniServant.(txbasic.TransactionServant), txUni)
		if err != nil {
			log.Errorf("Gas estimates error: %v", err)
			return txbasic.VerifyResult_Reject
		}
		balVal, err := txUniServant.GetBalance(tpcrtypes.NewFromBytes(txUni.Head.FeePayer), currency.TokenSymbol_Native)
		if err != nil {
			log.Errorf("Can't get payer %s balance: %v", tpcrtypes.NewFromBytes(txUni.Head.FeePayer), err)
			return txbasic.VerifyResult_Reject
		}
		if estiGas.Cmp(balVal) > 0 {
			log.Errorf("Insufficient gas: estiGas %s > balVal %s", estiGas.String(), balVal.String())
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalNonceVerifier() TransactionUniversalVerifier {
	return func(ctx context.Context, log tplog.Logger, nodeID string, txUni *TransactionUniversalWithHead, txUniServant TransactionUniversalServant) txbasic.VerifyResult {
		latestNonce, err := txUniServant.GetNonce(tpcrtypes.NewFromBytes(txUni.FromAddr))
		if err != nil {
			log.Errorf("Can't get from address %s nonce: %v", tpcrtypes.NewFromBytes(txUni.FromAddr), err)
			return txbasic.VerifyResult_Reject
		}
		if txUni.Nonce != latestNonce+1 {
			log.Errorf("Invalid txUni nonce: expected %d, actual %d", latestNonce+1, txUni.Nonce)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalPayerSignatureVerifier() TransactionUniversalVerifier {
	return func(ctx context.Context, log tplog.Logger, nodeID string, txUni *TransactionUniversalWithHead, txUniServant TransactionUniversalServant) txbasic.VerifyResult {
		cryType, err := tpcrtypes.NewFromBytes(txUni.Head.FeePayer).CryptType()
		if err != nil {
			log.Errorf("Can't get from address %s crypt type: %v", tpcrtypes.NewFromBytes(txUni.Head.FeePayer), err)
			return txbasic.VerifyResult_Reject
		}

		var signInfo tpcrtypes.SignatureInfo
		if err := json.Unmarshal(txUni.Head.FeePayerSignature, &signInfo); err != nil {
			return txbasic.VerifyResult_Reject
		}
		cryService, _ := txUniServant.GetCryptService(log, cryType)
		if ok, err := cryService.Verify(tpcrtypes.NewFromBytes(txUni.Head.FeePayer), txUni.Data.Specification, signInfo.SignData); !ok {
			log.Errorf("Can't verify payer signature: %v", err)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func ApplyTransactionUniversalVerifiers(ctx context.Context, log tplog.Logger, nodeID string, txUni *TransactionUniversalWithHead, txUniServant TransactionUniversalServant, verifiers ...TransactionUniversalVerifier) txbasic.VerifyResult {
	vrResult := txbasic.VerifyResult_Accept
	for _, verifier := range verifiers {
		vR := verifier(ctx, log, nodeID, txUni, txUniServant)
		switch vR {
		case txbasic.VerifyResult_Reject:
			return txbasic.VerifyResult_Reject
		case txbasic.VerifyResult_Ignore:
			vrResult = vR
		}
	}

	return vrResult
}
