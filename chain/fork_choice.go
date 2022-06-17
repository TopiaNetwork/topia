package chain

import (
	"bytes"
	"fmt"
	"math/big"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
)

type ForkChoice interface {
	BlockHeadChoice(bh1 *tpchaintypes.BlockHead, bh2 *tpchaintypes.BlockHead) (*tpchaintypes.BlockHead, error)

	FindAncestor(chain1 []*tpchaintypes.Block, chain2 []*tpchaintypes.Block) (*tpchaintypes.Block, bool)

	ChainChoiceAfterAncestor(chain1 []*tpchaintypes.Block, chain2 []*tpchaintypes.Block) ([]*tpchaintypes.Block, error)
}

func NewForChoice(log tplog.Logger, ledger ledger.Ledger, config *configuration.Configuration) ForkChoice {
	return &forkChoice{
		log:    log,
		ledger: ledger,
		config: config,
	}
}

type forkChoice struct {
	log    tplog.Logger
	ledger ledger.Ledger
	config *configuration.Configuration
}

func (fc *forkChoice) BlockHeadChoice(bh1 *tpchaintypes.BlockHead, bh2 *tpchaintypes.BlockHead) (*tpchaintypes.BlockHead, error) {
	if bh1.Height != bh1.Height {
		return nil, fmt.Errorf("Not same height block head: bh1 %d, bh2 %d", bh1.Height, bh2.Height)
	}

	/*
		wRatio1, err := ComputeNodeWeightRatio(fc.log, fc.ledger, bh1.Height, string(bh1.Proposer), fc.config.CSConfig.BlocksPerEpoch)
		if err != nil {
			return nil, err
		}
		wRatio2, err := ComputeNodeWeightRatio(fc.log, fc.ledger, bh2.Height, string(bh2.Proposer), fc.config.CSConfig.BlocksPerEpoch)
		if err != nil {
			return nil, err
		}
		if wRatio1 < wRatio2 {
			return bh2, nil
		} else if wRatio1 > wRatio2 {
			return bh1, nil
		}
	*/

	maxPriCMPR := new(big.Int).SetBytes(bh1.MaxPri).Cmp(new(big.Int).SetBytes(bh2.MaxPri))
	if maxPriCMPR < 0 {
		return bh2, nil
	} else if maxPriCMPR > 0 {
		return bh1, nil
	}

	/*
		hashCMPR := new(big.Int).SetBytes(bh1.Hash).Cmp(new(big.Int).SetBytes(bh2.Hash))
		if hashCMPR < 0 {
			return bh2, nil
		} else if hashCMPR > 0 {
			return bh1, nil
		}
	*/

	return bh1, nil
}

//make sure that chain1 and chain1 are both chained continuously
func (fc *forkChoice) FindAncestor(chain1 []*tpchaintypes.Block, chain2 []*tpchaintypes.Block) (*tpchaintypes.Block, bool) {
	cLen1 := len(chain1)
	cLen2 := len(chain2)

	if cLen1 == 0 || cLen2 == 0 {
		return nil, false
	}

	ancestorBLock := (*tpchaintypes.Block)(nil)

	i := cLen1 - 1
	j := cLen2 - 1
	for i >= 0 && j >= 0 {
		cbHashBytes1, _ := chain1[i].HashBytes()
		cbHashBytes2, _ := chain2[j].HashBytes()
		if bytes.Equal(cbHashBytes1, cbHashBytes2) {
			i--
			j--
		} else {
			if i < cLen1-1 {
				ancestorBLock = chain1[i+1]
			}
			break
		}
	}
	if ancestorBLock != nil {
		return ancestorBLock, true
	}

	return nil, false
}

//make sure that chain1 and chain1 are both chained continuously
func (fc *forkChoice) ChainChoiceAfterAncestor(chain1 []*tpchaintypes.Block, chain2 []*tpchaintypes.Block) ([]*tpchaintypes.Block, error) {
	cLen1 := len(chain1)
	cLen2 := len(chain2)

	if cLen1 == 0 {
		return chain2, nil
	}
	if cLen2 == 0 {
		return chain1, nil
	}

	i := 0
	wTotalRadio1 := 0.0
	wTotalRadio2 := 0.0

	maxLen := cLen1
	if cLen1 < cLen2 {
		maxLen = cLen2
	}
	for i < maxLen {
		if i <= cLen1 {
			wRatio1, err := ComputeNodeWeightRatio(fc.log, fc.ledger, chain1[i].Head.Height, string(chain1[i].Head.Proposer), fc.config.CSConfig.BlocksPerEpoch)
			if err != nil {
				return nil, err
			}
			wTotalRadio1 += wRatio1
		}

		if i < cLen2 {
			wRatio2, err := ComputeNodeWeightRatio(fc.log, fc.ledger, chain2[i].Head.Height, string(chain2[i].Head.Proposer), fc.config.CSConfig.BlocksPerEpoch)
			if err != nil {
				return nil, err
			}
			wTotalRadio2 += wRatio2
		}
	}

	if wTotalRadio1 < wTotalRadio2 {
		return chain2, nil
	}

	return chain1, nil
}
