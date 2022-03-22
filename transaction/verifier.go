package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	tpcmm "github.com/TopiaNetwork/topia/chain"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
)

type VerifyResult byte

const (
	ValidationResult_Unknown VerifyResult = iota
	VerifyResult_Accept
	VerifyResult_Reject
	VerifyResult_Ignore
)

type TransactionVerifier func(ctx context.Context, log tplog.Logger, tx *Transaction, txServant TansactionServant) VerifyResult

func TransactionChainIDVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction, txServant TansactionServant) VerifyResult {
		chainID := txServant.ChainID()
		if bytes.Compare(tx.ChainID, []byte(chainID)) != 0 {
			log.Errorf("Invalid chain ID: expected %s, actual %s", chainID, string(tx.ChainID))
			return VerifyResult_Reject
		}

		return VerifyResult_Accept
	}
}

func TransactionAddressVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction, txServant TansactionServant) VerifyResult {
		fromAddr := tpcrtypes.NewFromBytes(tx.FromAddr)
		targetAddr := tpcrtypes.NewFromBytes(tx.TargetAddr)
		payerAddr := tpcrtypes.NewFromBytes(tx.FeePayer)

		fEth := fromAddr.IsEth()
		tEth := targetAddr.IsEth()
		pEth := payerAddr.IsEth()

		if fEth && tEth && pEth {
			return VerifyResult_Accept
		} else if !fEth && !tEth && !pEth {
			fromAddrType, err := fromAddr.CryptType()
			if err != nil {
				log.Errorf("Can't get from address type: %v", err)
				return VerifyResult_Reject
			}

			if isValid, _ := fromAddr.IsValid(tpnet.CurrentNetworkType, fromAddrType); !isValid {
				log.Errorf("Invalid from address: %v", tx.FromAddr)
				return VerifyResult_Reject
			}

			if isValid, _ := targetAddr.IsValid(tpnet.CurrentNetworkType, fromAddrType); !isValid {
				log.Errorf("Invalid target address: %v", tx.TargetAddr)
				return VerifyResult_Reject
			}

			if isValid, _ := fromAddr.IsValid(tpnet.CurrentNetworkType, fromAddrType); !isValid {
				log.Errorf("Invalid payer address: %v", tx.FeePayer)
				return VerifyResult_Reject
			}

			return VerifyResult_Accept
		} else {
			log.Errorf("Invliad tx address: fEth=%v, tEth=%v, pEth=%v", fEth, tEth, pEth)
			return VerifyResult_Reject
		}
	}
}

func TransactionGasVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction, txServant TansactionServant) VerifyResult {
		if tx.GasPrice < txServant.GetGasConfig().MinGasPrice {
			log.Errorf("GasPrice %s less than the min gas price %d", tx.GasPrice, txServant.GetGasConfig().MinGasPrice)
			return VerifyResult_Reject
		}

		if tx.GasLimit < txServant.GetGasConfig().MinGasLimit {
			log.Errorf("GasLimit %s less than the min gas limit %d", tx.GasPrice, txServant.GetGasConfig().MinGasLimit)
			return VerifyResult_Reject
		}

		gasEstimator, err := txServant.GetGasEstimator()
		if err != nil {
			log.Errorf("Can't get gas estimator: %v", err)
			return VerifyResult_Reject
		}
		estiGas, err := gasEstimator.Estimate(tx)
		if err != nil {
			log.Errorf("Gas estimates error: %v", err)
			return VerifyResult_Reject
		}
		balVal, err := txServant.GetBalance(tpcmm.TokenSymbol_Native, tpcrtypes.NewFromBytes(tx.FeePayer))
		if err != nil {
			log.Errorf("Can't get payer %s balance: %v", tpcrtypes.NewFromBytes(tx.FeePayer), err)
			return VerifyResult_Reject
		}
		if estiGas.Cmp(balVal) > 0 {
			log.Errorf("Insufficient gas: estiGas %s > balVal %s", estiGas.String(), balVal.String())
			return VerifyResult_Reject
		}

		return VerifyResult_Accept
	}
}

func TransactionNonceVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction, txServant TansactionServant) VerifyResult {
		latestNonce, err := txServant.GetNonce(tpcrtypes.NewFromBytes(tx.FromAddr))
		if err != nil {
			log.Errorf("Can't get from address %s nonce: %v", tpcrtypes.NewFromBytes(tx.FromAddr), err)
			return VerifyResult_Reject
		}
		if tx.Nonce != latestNonce+1 {
			log.Errorf("Invalid tx nonce: expected %d, actual %d", latestNonce+1, tx.Nonce)
			return VerifyResult_Reject
		}

		return VerifyResult_Accept
	}
}

func TransactionSignatureVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction, txServant TansactionServant) VerifyResult {
		cryType, err := tpcrtypes.NewFromBytes(tx.FromAddr).CryptType()
		if err != nil {
			log.Errorf("Can't get from address %s crypt type: %v", tpcrtypes.NewFromBytes(tx.FromAddr), err)
			return VerifyResult_Reject
		}

		cryService, _ := txServant.GetCryptService(log, cryType)

		var fromSign tpcrtypes.SignatureInfo
		err = json.Unmarshal(tx.Signature, &fromSign)
		if err != nil {
			log.Errorf("Can't unmarshal tx signature: %v", err)
			return VerifyResult_Reject
		}
		if ok, err := cryService.Verify(fromSign.PublicKey, tx.Data, fromSign.SignData); !ok {
			log.Errorf("Can't verify tx signature: %v", err)
			return VerifyResult_Reject
		}

		var payerSign tpcrtypes.SignatureInfo
		err = json.Unmarshal(tx.Signature, &payerSign)
		if err != nil {
			log.Errorf("Can't unmarshal payer signature: %v", err)
			return VerifyResult_Reject
		}
		if ok, err := cryService.Verify(payerSign.PublicKey, tx.Data, payerSign.SignData); !ok {
			log.Errorf("Can't verify payer signature: %v", err)
			return VerifyResult_Reject
		}

		return VerifyResult_Accept
	}
}

func ApplyTransactionVerifiers(ctx context.Context, log tplog.Logger, tx *Transaction, txServant TansactionServant, verifiers ...TransactionVerifier) VerifyResult {
	vrResult := VerifyResult_Accept
	for _, verifier := range verifiers {
		vR := verifier(ctx, log, tx, txServant)
		switch vR {
		case VerifyResult_Reject:
			return VerifyResult_Reject
		case VerifyResult_Ignore:
			vrResult = vR
		}
	}

	return vrResult
}
