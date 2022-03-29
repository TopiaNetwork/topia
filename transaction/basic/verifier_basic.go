package basic

import (
	"bytes"
	"context"
	"encoding/json"

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

type TransactionVerifier func(ctx context.Context, log tplog.Logger, txI interface{}, txServant TansactionServant) VerifyResult

func TransactionChainIDVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, txI interface{}, txServant TansactionServant) VerifyResult {
		tx := txI.(*TransactionHead)
		chainID := txServant.ChainID()
		if bytes.Compare(tx.ChainID, []byte(chainID)) != 0 {
			log.Errorf("Invalid chain ID: expected %s, actual %s", chainID, string(tx.ChainID))
			return VerifyResult_Reject
		}

		return VerifyResult_Accept
	}
}

func TransactionFromAddressVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, txI interface{}, txServant TansactionServant) VerifyResult {
		tx := txI.(*TransactionHead)
		fromAddr := tpcrtypes.NewFromBytes(tx.FromAddr)

		fEth := fromAddr.IsEth()

		if fEth && string(tx.Category) == TransactionCategory_Eth {
			return VerifyResult_Accept
		} else if !fEth {
			cryType, err := fromAddr.CryptType()
			if err != nil {
				log.Errorf("Can't get from address type: %v", err)
				return VerifyResult_Reject
			}

			if isValid, _ := fromAddr.IsValid(tpnet.CurrentNetworkType, cryType); !isValid {
				log.Errorf("Invalid from address: %v", tx.FromAddr)
				return VerifyResult_Reject
			}

			return VerifyResult_Accept
		} else {
			log.Errorf("Invliad tx address: fEth=%v", fEth)
			return VerifyResult_Reject
		}
	}
}

func TransactionSignatureVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, txI interface{}, txServant TansactionServant) VerifyResult {
		tx := txI.(*Transaction)

		cryType, err := tx.CryptType()
		if err != nil {
			log.Errorf("Can't get from address %s crypt type: %v", tpcrtypes.NewFromBytes(tx.Head.FromAddr), err)
			return VerifyResult_Reject
		}

		cryService, _ := txServant.GetCryptService(log, cryType)

		var fromSign tpcrtypes.SignatureInfo
		err = json.Unmarshal(tx.Head.Signature, &fromSign)
		if err != nil {
			log.Errorf("Can't unmarshal tx signature: %v", err)
			return VerifyResult_Reject
		}
		if ok, err := cryService.Verify(fromSign.PublicKey, tx.Data.Specification, fromSign.SignData); !ok {
			log.Errorf("Can't verify tx signature: %v", err)
			return VerifyResult_Reject
		}

		return VerifyResult_Accept
	}
}

func ApplyTransactionVerifiers(ctx context.Context, log tplog.Logger, txT interface{}, txServant TansactionServant, verifiers ...TransactionVerifier) VerifyResult {
	vrResult := VerifyResult_Accept
	for _, verifier := range verifiers {
		vR := verifier(ctx, log, txT, txServant)
		switch vR {
		case VerifyResult_Reject:
			return VerifyResult_Reject
		case VerifyResult_Ignore:
			vrResult = vR
		}
	}

	return vrResult
}
