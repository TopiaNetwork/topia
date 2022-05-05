package contract

import (
	"context"
	"github.com/TopiaNetwork/topia/account"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type ContractAccount struct {
}

func (a *ContractAccount) BindName(ctx context.Context, addr tpcrtypes.Address, accountName account.AccountName) error {
	/*fromAddr, ok := ctx.Value(tpvmcmm.VMCtxKey_FromAddr).(tpcrtypes.Address)
	if !ok {
		return errors.New("There is no from address info in vm context")
	}*/

	return nil
}
