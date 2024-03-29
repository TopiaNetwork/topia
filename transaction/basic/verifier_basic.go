package basic

import (
	"bytes"
	"context"
	"encoding/json"
	tpcmm "github.com/TopiaNetwork/topia/common"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type VerifyResult byte

const (
	ValidationResult_Unknown VerifyResult = iota
	VerifyResult_Accept
	VerifyResult_Reject
	VerifyResult_Ignore
)

type TransactionVerifier func(ctx context.Context, log tplog.Logger, txI interface{}, txServant TransactionServant) VerifyResult

func TransactionChainIDVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, txI interface{}, txServant TransactionServant) VerifyResult {
		tx, ok := txI.(*Transaction)
		if !ok {
			log.Panicf("Invalid txI, expected Transaction")
		}
		chainID := txServant.ChainID()
		if bytes.Compare(tx.Head.ChainID, []byte(chainID)) != 0 {
			log.Errorf("Invalid chain ID: expected %s, actual %s", chainID, string(tx.Head.ChainID))
			return VerifyResult_Reject
		}

		return VerifyResult_Accept
	}
}

func TransactionFromAddressVerifier() TransactionVerifier {
	return func(ctx context.Context, log tplog.Logger, txI interface{}, txServant TransactionServant) VerifyResult {
		tx, ok := txI.(*Transaction)
		if !ok {
			log.Panicf("Invalid txI, expected Transaction")
		}

		fromAddr := tpcrtypes.NewFromBytes(tx.Head.FromAddr)

		fEth := tpcrtypes.IsEth(string(fromAddr))

		if fEth && string(tx.Head.Category) == TransactionCategory_Eth {
			return VerifyResult_Accept
		} else if !fEth {
			if isValid := fromAddr.IsValid(tpcmm.CurrentNetworkType); !isValid {
				log.Errorf("Invalid from address: %v", string(tx.Head.FromAddr))
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
	return func(ctx context.Context, log tplog.Logger, txI interface{}, txServant TransactionServant) VerifyResult {
		tx, ok := txI.(*Transaction)
		if !ok {
			log.Panicf("Invalid txI, expected Transaction")
		}

		cryType, err := tx.CryptType()
		if err != nil {
			log.Errorf("Can't get from address %s crypt type: %v", tpcrtypes.NewFromBytes(tx.Head.FromAddr), err)
			return VerifyResult_Reject
		}

		var signInfo tpcrtypes.SignatureInfo
		if err := json.Unmarshal(tx.Head.Signature, &signInfo); err != nil {
			return VerifyResult_Reject
		}
		cryService, _ := txServant.GetCryptService(log, cryType)
		if ok, err := cryService.Verify(tpcrtypes.NewFromBytes(tx.Head.FromAddr), tx.Data.Specification, signInfo.SignData); !ok {
			log.Errorf("Can't verify tx signature: %v", err)
			return VerifyResult_Reject
		}

		return VerifyResult_Accept
	}
}

func ApplyTransactionVerifiers(ctx context.Context, log tplog.Logger, txT interface{}, txServant TransactionServant, verifiers ...TransactionVerifier) VerifyResult {
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
