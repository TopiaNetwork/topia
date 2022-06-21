package chain

import (
	"fmt"
	"math/big"

	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

func ComputeNodeWeightRatio(log tplog.Logger, ledger ledger.Ledger, height uint64, nodeID string, blocksPerEpoch uint64) (float64, error) {
	compStateRN := state.CreateCompositionStateReadonlyAt(log, ledger, height)
	if compStateRN == nil {
		return 0, fmt.Errorf("Can't get composition read only state: version %d, node %s", height, nodeID)
	}
	defer compStateRN.Stop()

	totalWeight, _ := compStateRN.GetTotalWeight()
	nodeWeight, _ := compStateRN.GetNodeWeight(nodeID)

	rWeight := new(big.Rat).SetFloat64(float64(totalWeight))
	fWeightRatio, _ := new(big.Rat).Quo(new(big.Rat).SetInt64(int64(nodeWeight*blocksPerEpoch)), rWeight).Float64()

	return fWeightRatio, nil
}




