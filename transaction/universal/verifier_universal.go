package universal

import (
	"context"
	"encoding/json"

	tpcmm "github.com/TopiaNetwork/topia/chain"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionUniversalVerifier func(ctx context.Context, log tplog.Logger, txUni *TransactionUniversalWithHead, txUniServant TansactionUniversalServant) txbasic.VerifyResult

func TransactionUniversalPayerAddressVerifier() TransactionUniversalVerifier {
	return func(ctx context.Context, log tplog.Logger, txUni *TransactionUniversalWithHead, txUniServant TansactionUniversalServant) txbasic.VerifyResult {
		payerAddr := tpcrtypes.NewFromBytes(txUni.Head.FeePayer)

		fromAddrType, err := payerAddr.CryptType()
		if err != nil {
			log.Errorf("Can't get from address type: %v", err)
			return txbasic.VerifyResult_Reject
		}

		if isValid, _ := payerAddr.IsValid(tpnet.CurrentNetworkType, fromAddrType); !isValid {
			log.Errorf("Invalid payer address: %v", txUni.Head.FeePayer)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalGasVerifier() TransactionUniversalVerifier {
	return func(ctx context.Context, log tplog.Logger, txUni *TransactionUniversalWithHead, txUniServant TansactionUniversalServant) txbasic.VerifyResult {
		if txUni.Head.GasPrice < txUniServant.GetGasConfig().MinGasPrice {
			log.Errorf("GasPrice %s less than the min gas price %d", txUni.Head.GasPrice, txUniServant.GetGasConfig().MinGasPrice)
			return txbasic.VerifyResult_Reject
		}

		if txUni.Head.GasLimit < txUniServant.GetGasConfig().MinGasLimit {
			log.Errorf("GasLimit %s less than the min gas limit %d", txUni.Head.GasPrice, txUniServant.GetGasConfig().MinGasLimit)
			return txbasic.VerifyResult_Reject
		}

		gasEstimator, err := txUniServant.GetGasEstimator()
		if err != nil {
			log.Errorf("Can't get gas estimator: %v", err)
			return txbasic.VerifyResult_Reject
		}
		estiGas, err := gasEstimator.Estimate(txUni)
		if err != nil {
			log.Errorf("Gas estimates error: %v", err)
			return txbasic.VerifyResult_Reject
		}
		balVal, err := txUniServant.GetBalance(tpcmm.TokenSymbol_Native, tpcrtypes.NewFromBytes(txUni.Head.FeePayer))
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
	return func(ctx context.Context, log tplog.Logger, txUni *TransactionUniversalWithHead, txUniServant TansactionUniversalServant) txbasic.VerifyResult {
		latestNonce, err := txUniServant.GetNonce(tpcrtypes.NewFromBytes(txUni.FromAddr))
		if err != nil {
			log.Errorf("Can't get from address %s nonce: %v", tpcrtypes.NewFromBytes(txUni.FromAddr), err)
			return txbasic.VerifyResult_Reject
		}
		if txUni.Head.Nonce != latestNonce+1 {
			log.Errorf("Invalid txUni nonce: expected %d, actual %d", latestNonce+1, txUni.Head.Nonce)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func TransactionUniversalPayerSignatureVerifier() TransactionUniversalVerifier {
	return func(ctx context.Context, log tplog.Logger, txUni *TransactionUniversalWithHead, txUniServant TansactionUniversalServant) txbasic.VerifyResult {
		cryType, err := tpcrtypes.NewFromBytes(txUni.Head.FeePayer).CryptType()
		if err != nil {
			log.Errorf("Can't get from address %s crypt type: %v", tpcrtypes.NewFromBytes(txUni.Head.FeePayer), err)
			return txbasic.VerifyResult_Reject
		}

		cryService, _ := txUniServant.GetCryptService(log, cryType)

		var fromSign tpcrtypes.SignatureInfo
		err = json.Unmarshal(txUni.Signature, &fromSign)
		if err != nil {
			log.Errorf("Can't unmarshal txUni signature: %v", err)
			return txbasic.VerifyResult_Reject
		}
		if ok, err := cryService.Verify(fromSign.PublicKey, txUni.Data.Specification, fromSign.SignData); !ok {
			log.Errorf("Can't verify txUni signature: %v", err)
			return txbasic.VerifyResult_Reject
		}

		var payerSign tpcrtypes.SignatureInfo
		err = json.Unmarshal(txUni.Signature, &payerSign)
		if err != nil {
			log.Errorf("Can't unmarshal payer signature: %v", err)
			return txbasic.VerifyResult_Reject
		}
		if ok, err := cryService.Verify(payerSign.PublicKey, txUni.Data.Specification, payerSign.SignData); !ok {
			log.Errorf("Can't verify payer signature: %v", err)
			return txbasic.VerifyResult_Reject
		}

		return txbasic.VerifyResult_Accept
	}
}

func ApplyTransactionUniversalVerifiers(ctx context.Context, log tplog.Logger, txUni *TransactionUniversalWithHead, txUniServant TansactionUniversalServant, verifiers ...TransactionUniversalVerifier) txbasic.VerifyResult {
	vrResult := txbasic.VerifyResult_Accept
	for _, verifier := range verifiers {
		vR := verifier(ctx, log, txUni, txUniServant)
		switch vR {
		case txbasic.VerifyResult_Reject:
			return txbasic.VerifyResult_Reject
		case txbasic.VerifyResult_Ignore:
			vrResult = vR
		}
	}

	return vrResult
}
