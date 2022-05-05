package contract

import (
	"context"
	"fmt"

	"github.com/TopiaNetwork/topia/account"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
)

type ContractAccount struct {
}

func (a *ContractAccount) BindName(ctx context.Context, addr tpcrtypes.Address, accountName account.AccountName) error {
	cCtx := tpvmcmm.NewContractContext(ctx)

	fromAddr, err := cCtx.GetFromAccount()
	if err != nil {
		return err
	}

	if !accountName.IsChild(fromAddr.Name) {
		return fmt.Errorf("Bind account name is not child  of from account: bind addr %d name %s, from addr %d name %s")
	}

	vmServant, err := cCtx.GetServant()
	if err != nil {
		return fmt.Errorf("Can't get vm servant: err %v", err)
	}

	return vmServant.UpdateName(addr, accountName)

	return nil
}
