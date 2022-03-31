package transaction

import (
	"context"

	tplog "github.com/TopiaNetwork/topia/log"
)

type ValidationResult byte

const (
	ValidationResult_Unknown ValidationResult = iota
	ValidationResult_Accecpt
	ValidationResult_Reject
	ValidationResult_Ignore
)

type TransactionValidator func(ctx context.Context, log tplog.Logger, tx *Transaction) ValidationResult

func TransactionValidatorWithGas() TransactionValidator {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction) ValidationResult {
		panic("implement me")
	}
}

func TransactionValidatorWithBalance() TransactionValidator {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction) ValidationResult {
		panic("implement me")
	}
}

func TransactionValidatorWithAddress() TransactionValidator {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction) ValidationResult {
		panic("implement me")
	}
}

func TransactionValidatorWithNonce() TransactionValidator {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction) ValidationResult {
		panic("implement me")
	}
}

func TransactionValidatorWithSignature() TransactionValidator {
	return func(ctx context.Context, log tplog.Logger, tx *Transaction) ValidationResult {
		panic("implement me")
	}
}

func ApplyTransactionValidator(ctx context.Context, log tplog.Logger, tx *Transaction, validators ...TransactionValidator) ValidationResult {
	valResult := ValidationResult_Accecpt
	for _, validator := range validators {
		valR := validator(ctx, log, tx)
		switch valResult {
		case ValidationResult_Reject:
			return ValidationResult_Reject
		case ValidationResult_Ignore:
			valResult = valR
		}
	}

	return valResult
}
