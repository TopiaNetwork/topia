package chain

import (
	"github.com/Gurpartap/async"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
)

type blockValidator struct {
}

func (v *blockValidator) validateBase(bh *tpchaintypes.BlockHead) error {
	return nil
}

func (v *blockValidator) ValidateHeader(bh *tpchaintypes.BlockHead) error {
	async.Err(func() error {
		return nil
	})

	return nil
}
