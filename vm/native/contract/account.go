package contract

import (
	"context"
	"fmt"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
)

type ContractAccount struct {
}

func (a *ContractAccount) BindName(ctx context.Context, parentAddr tpcrtypes.Address, addr tpcrtypes.Address, accountName tpacc.AccountName) error {
	cCtx := tpvmcmm.NewContractContext(ctx)

	vmServant, err := cCtx.GetServant()
	if err != nil {
		return fmt.Errorf("Can't get vm servant: err %v", err)
	}
	parentAcc, err := vmServant.GetAccount(parentAddr)
	if err != nil {
		return fmt.Errorf("Can't get parent account: err %v", err)
	}

	if !accountName.IsChild(parentAcc.Name) {
		return fmt.Errorf("Bind account name is not child of account: bind addr %d name %s, parent addr %d name %s", addr, accountName, parentAddr, parentAcc.Name)
	}

	gasUsed := Account_BindName + vmServant.GetGasConfig().GasEachByte*uint64(len(accountName))
	if err = vmServant.GasUsedAccumulate(gasUsed); err != nil {
		return err
	}

	return vmServant.UpdateName(addr, accountName)
}

func (a *ContractAccount) grantAccessOperation(ctx context.Context, parentAddr tpcrtypes.Address, addr tpcrtypes.Address, operaGas uint64, operation func(acc *tpacc.Account) (uint64, error)) error {
	cCtx := tpvmcmm.NewContractContext(ctx)

	vmServant, err := cCtx.GetServant()
	if err != nil {
		return fmt.Errorf("Can't get vm servant: err %v", err)
	}
	parentAcc, err := vmServant.GetAccount(parentAddr)
	if err != nil {
		return fmt.Errorf("Can't get parent account: err %v, addr %s", err, parentAddr)
	}

	acc, err := vmServant.GetAccount(parentAddr)
	if err != nil {
		return fmt.Errorf("Can't get account: err %v, addr %s", err, addr)
	}

	if !acc.Name.IsChild(parentAcc.Name) {
		return fmt.Errorf("The account name is not child of account: addr %d name %s, parent addr %d name %s", addr, acc.Name, parentAddr, parentAcc.Name)
	}

	usedSize, err := operation(acc)
	if err != nil {
		return err
	}

	gasUsed := operaGas + vmServant.GetGasConfig().GasEachByte*usedSize
	if err = vmServant.GasUsedAccumulate(gasUsed); err != nil {
		return err
	}

	return vmServant.UpdateAccount(acc)
}

func (a *ContractAccount) GrantAccessMethods(ctx context.Context, parentAddr tpcrtypes.Address, addr tpcrtypes.Address, contractAddr tpcrtypes.Address, methods []string, gasLimit uint64) error {
	return a.grantAccessOperation(ctx, parentAddr, addr, Account_GrantAccessMethods, func(acc *tpacc.Account) (uint64, error) {
		accToken := acc.Token
		usedSize := 0
		if accToken == nil {
			perm := tpacc.NewPermissionContractMethod(gasLimit)
			accToken = tpacc.NewAccountToken(perm)

			acc.Token = accToken

			usedSize = 2
		}

		usedSize += len(contractAddr)

		for _, method := range methods {
			acc.Token.Permission.AddMethod(contractAddr, method)
			usedSize += len(method)
		}

		return uint64(usedSize), nil
	})
}

func (a *ContractAccount) GrantAccessRoot(ctx context.Context, parentAddr tpcrtypes.Address, addr tpcrtypes.Address) error {
	return a.grantAccessOperation(ctx, parentAddr, addr, Account_GrantAccessRoot, func(acc *tpacc.Account) (uint64, error) {
		accToken := acc.Token
		usedSize := uint64(0)
		if accToken == nil {
			perm := tpacc.NewPermissionRoot()
			accToken = tpacc.NewAccountToken(perm)

			acc.Token = accToken

			usedSize = 2
		}

		return usedSize, nil
	})
}

func (a *ContractAccount) RevokeGrantAccess(ctx context.Context, parentAddr tpcrtypes.Address, addr tpcrtypes.Address) error {
	return a.grantAccessOperation(ctx, parentAddr, addr, Account_RevokeGrantAccess, func(acc *tpacc.Account) (uint64, error) {
		accToken := acc.Token
		if accToken != nil {
			acc.Token = nil
		}

		return 0, nil
	})
}

func (a *ContractAccount) RevokeGrantAccessMethods(ctx context.Context, parentAddr tpcrtypes.Address, addr tpcrtypes.Address, contractAddr tpcrtypes.Address, methods []string) error {
	return a.grantAccessOperation(ctx, parentAddr, addr, Account_RevokeGrantAccessMethods, func(acc *tpacc.Account) (uint64, error) {
		accToken := acc.Token
		if accToken != nil {
			for _, method := range methods {
				acc.Token.Permission.RemoveMethod(contractAddr, method)
			}
		}

		return 0, nil
	})
}
