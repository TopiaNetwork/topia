package common

import (
	"context"
	"fmt"
	"reflect"
	"unsafe"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type ContractContext struct {
	context.Context
}

func NewContractContext(ctx context.Context) *ContractContext {
	return &ContractContext{
		Context: ctx,
	}
}

func (cc *ContractContext) GetCtxValues(ctxKeys []VMCtxKey, ctxValues []unsafe.Pointer) error {
	for i, ctxKey := range ctxKeys {
		switch ctxKeyVal := cc.Value(ctxKey).(type) {
		case VMServant:
			*(*VMServant)(ctxValues[i]) = ctxKeyVal
		case *tpcrtypes.Address:
			*(**tpcrtypes.Address)(ctxValues[i]) = ctxKeyVal
		default:
			return fmt.Errorf("Invalid value type: key %s, value type %s", ctxKey.String(), reflect.TypeOf(ctxKeyVal).String())
		}
	}

	return nil
}

func (cc *ContractContext) GetServant() (VMServant, error) {
	var vmServant VMServant

	err := cc.GetCtxValues([]VMCtxKey{VMCtxKey_VMServant}, []unsafe.Pointer{unsafe.Pointer(&vmServant)})
	if err != nil {
		return nil, err
	}

	return vmServant, nil
}

func (cc *ContractContext) GetFromAccount() (*tpacc.Account, error) {
	var vmServant VMServant
	var fromAddr *tpcrtypes.Address
	err := cc.GetCtxValues([]VMCtxKey{VMCtxKey_VMServant, VMCtxKey_FromAddr}, []unsafe.Pointer{unsafe.Pointer(&vmServant), unsafe.Pointer(&fromAddr)})
	if err != nil {
		return nil, err
	}

	return vmServant.GetAccount(*fromAddr)
}
