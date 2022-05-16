package service

import (
	"context"
	"fmt"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type AccountService interface {
	BindName(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address, accountName tpacc.AccountName) (txbasic.TxID, error)

	GrantAccessRoot(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address) (txbasic.TxID, error)

	GrantAccessMethods(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address, contractAddr tpcrtypes.Address, methods []string, gasAllowance uint64) (txbasic.TxID, error)

	RevokeGrantAccess(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address) (txbasic.TxID, error)

	RevokeGrantAccessMethods(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address, contractAddr tpcrtypes.Address, methods []string) (txbasic.TxID, error)
}

type accountService struct {
	log             tplog.Logger
	contractService ContractService
}

func NewAccountService(log tplog.Logger, contractService ContractService) AccountService {
	return &accountService{
		log:             log,
		contractService: contractService,
	}
}

func (as *accountService) BindName(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address, accountName tpacc.AccountName) (txbasic.TxID, error) {
	args := fmt.Sprintf("%s@%s@%s", fromAddr, accountAddr, accountName)

	txID, _, err := as.contractService.Invoke(ctx, fromAddr, payerAddr, gasPrice, gasLimit, tpacc.NativeContractAccount_Account.Addr, "BindName", args)

	return txID, err
}

func (as *accountService) GrantAccessRoot(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address) (txbasic.TxID, error) {
	args := fmt.Sprintf("%s@%s", fromAddr, accountAddr)

	txID, _, err := as.contractService.Invoke(ctx, fromAddr, payerAddr, gasPrice, gasLimit, tpacc.NativeContractAccount_Account.Addr, "GrantAccessRoot", args)

	return txID, err
}

func (as *accountService) GrantAccessMethods(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address, contractAddr tpcrtypes.Address, methods []string, gasAllowance uint64) (txbasic.TxID, error) {
	args := fmt.Sprintf("%s@%s@%s@%v@%d", fromAddr, accountAddr, contractAddr, methods, gasAllowance)

	txID, _, err := as.contractService.Invoke(ctx, fromAddr, payerAddr, gasPrice, gasLimit, tpacc.NativeContractAccount_Account.Addr, "GrantAccessMethods", args)

	return txID, err
}

func (as *accountService) RevokeGrantAccess(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address) (txbasic.TxID, error) {
	args := fmt.Sprintf("%s@%s", fromAddr, accountAddr)

	txID, _, err := as.contractService.Invoke(ctx, fromAddr, payerAddr, gasPrice, gasLimit, tpacc.NativeContractAccount_Account.Addr, "RevokeGrantAccess", args)

	return txID, err
}

func (as *accountService) RevokeGrantAccessMethods(ctx context.Context, fromAddr tpcrtypes.Address, payerAddr tpcrtypes.Address, gasPrice uint64, gasLimit uint64, accountAddr tpcrtypes.Address, contractAddr tpcrtypes.Address, methods []string) (txbasic.TxID, error) {
	args := fmt.Sprintf("%s@%s@%s@%v", fromAddr, accountAddr, contractAddr, methods)

	txID, _, err := as.contractService.Invoke(ctx, fromAddr, payerAddr, gasPrice, gasLimit, tpacc.NativeContractAccount_Account.Addr, "RevokeGrantAccessMethods", args)

	return txID, err
}
