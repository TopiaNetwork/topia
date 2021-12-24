package transaction

import (
	"context"
	"fmt"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type ReturnCode int

const (
	ReturnCode_OK ReturnCode = iota
	ReturnCode_ValidationReject
	ReturnCode_ExeNotFound
)

func (rc ReturnCode) String() string {
	switch rc {
	case ReturnCode_OK:
		return "ReturnCode_OK"
	case ReturnCode_ValidationReject:
		return "ReturnCode_ValidationReject"
	case ReturnCode_ExeNotFound:
		return "ReturnCode_ExeNotFound"
	default:
		return fmt.Sprintf("unknown error, code: %d", rc)
	}
}

type TransactionExecutorAction interface {
	doExecute(ctx context.Context, tx *Transaction) ReturnCode
}

type TransactionExecutor struct {
	log           tplog.Logger
	validators    []TransactionValidator
	executeAction TransactionExecutorAction
}

func NewTransactionExecutor(level tplogcmm.LogLevel, log tplog.Logger) *TransactionExecutor {
	exLog := tplog.CreateModuleLogger(level, "TransactionExecutor", log)
	return &TransactionExecutor{
		log: exLog,
	}
}

func (exe *TransactionExecutor) ResetValidators(validators ...TransactionValidator) {
	exe.validators = []TransactionValidator{}
	for _, validator := range validators {
		exe.validators = append(exe.validators, validator)
	}
}

func (exe *TransactionExecutor) ResetexecuteAction(executeAction TransactionExecutorAction) {
	exe.executeAction = executeAction
}

func (exe *TransactionExecutor) Execute(ctx context.Context, tx *Transaction) ReturnCode {
	valResult := ApplyTransactionValidator(ctx, exe.log, tx)
	if valResult == ValidationResult_Reject {
		return ReturnCode_ValidationReject
	}

	var executeAction TransactionExecutorAction

	txType := tx.GetType()
	switch txType {
	case TransactionType_Transfer:
		executeAction = &TransactionExecutorActionTran{
			log: exe.log,
		}
	case TransactionType_NativeInvoke:
		executeAction = &TransactionExecutorActionBuildinInV{
			log: exe.log,
		}
	default:

	}
	if executeAction != nil {
		exe.ResetexecuteAction(executeAction)
		return exe.executeAction.doExecute(ctx, tx)
	} else {
		return ReturnCode_ExeNotFound
	}

	return exe.executeAction.doExecute(ctx, tx)
}
