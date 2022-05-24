package contract

import (
	"context"
	tpcmm "github.com/TopiaNetwork/topia/common"
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

func (ct *ContractTest) TestFuncWithStruct(ctx context.Context, NodeID string) (*tpcmm.NodeInfo, int, string, error) {
	return &tpcmm.NodeInfo{
		NodeID: NodeID,
		Weight: 1000,
		Role:   tpcmm.NodeRole_Executor,
		State:  tpcmm.NodeState_Active,
	}, 100, "ContractTest", nil
}
