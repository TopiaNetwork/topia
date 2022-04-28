package contract

import (
	"context"
	"errors"

	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/state"
)

type ContractTest struct {
}

func NewContractTest() *ContractTest {
	return &ContractTest{}
}

func (ct *ContractTest) TestFuncSimple(ctx context.Context, i int) (int, error) {
	i = i + 100
	return i, nil
}

func (ct *ContractTest) TestFuncWithStruct(ctx context.Context, NodeID string) (*tpcmm.NodeInfo, error) {
	compState := ctx.Value("COMPSTATE").(state.CompositionState)
	if compState == nil {
		return nil, errors.New("Can't get CompositionState from ctx")
	}

	return &tpcmm.NodeInfo{
		NodeID: NodeID,
		Weight: 1000,
		Role:   tpcmm.NodeRole_Executor,
		State:  tpcmm.NodeState_Active,
	}, nil
}