package consensus

import (
	"context"
	"fmt"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/integration/mock"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
)

func buildNodeConnections(networks []tpnet.Network) {
	if len(networks) <= 1 {
		return
	}

	for i := 1; i < len(networks); i++ {
		networks[i].Connect(networks[0].ListenAddr())
	}

	buildNodeConnections(networks[1:])
}

func createNetworkNodes(log tplog.Logger,
	sysActorExe []*actor.ActorSystem,
	sysActorProp []*actor.ActorSystem,
	sysActorVal []*actor.ActorSystem,
	activeNodesExe []*mock.NetworkActiveNodeMock,
	activeNodesProp []*mock.NetworkActiveNodeMock,
	activeNodesVal []*mock.NetworkActiveNodeMock,
	t *testing.T) ([]tpnet.Network, []tpnet.Network, []tpnet.Network) {
	var networkExes []tpnet.Network
	for i := 0; i < len(sysActorExe); i++ {
		network := tpnet.NewNetwork(context.Background(), log, sysActorExe[i], fmt.Sprintf("/ip4/127.0.0.1/tcp/4100%s", i), "topia1", activeNodesExe[i])
		log.Infof("Execute network %d id=%s", i, network.ID())
		networkExes = append(networkExes, network)

		activeNodesExe[i].AddActiveExecutor(network.ID())
		for _, activeNode := range activeNodesProp {
			activeNode.AddActiveExecutor(network.ID())
		}
		for _, activeNode := range activeNodesVal {
			activeNode.AddActiveExecutor(network.ID())
		}
	}

	var networkProps []tpnet.Network
	for i := 0; i < len(sysActorProp); i++ {
		network := tpnet.NewNetwork(context.Background(), log, sysActorProp[i], fmt.Sprintf("/ip4/127.0.0.1/tcp/5100%s", i), "topia1", activeNodesProp[i])
		log.Infof("Propose network %d id=%s", i, network.ID())
		networkProps = append(networkProps, network)

		activeNodesProp[i].AddActiveProposer(network.ID())
		for _, activeNode := range activeNodesExe {
			activeNode.AddActiveProposer(network.ID())
		}
		for _, activeNode := range activeNodesVal {
			activeNode.AddActiveProposer(network.ID())
		}
	}

	var networkVals []tpnet.Network
	for i := 0; i < len(sysActorVal); i++ {
		network := tpnet.NewNetwork(context.Background(), log, sysActorVal[i], fmt.Sprintf("/ip4/127.0.0.1/tcp/6100%s", i), "topia1", activeNodesVal[i])
		log.Infof("Validate network %d id=%s", i, network.ID())
		networkVals = append(networkVals, network)

		activeNodesVal[i].AddActiveValidator(network.ID())
		for _, activeNode := range activeNodesExe {
			activeNode.AddActiveValidator(network.ID())
		}
		for _, activeNode := range activeNodesProp {
			activeNode.AddActiveValidator(network.ID())
		}
	}

	buildNodeConnections(networkExes)
	buildNodeConnections(networkProps)
	buildNodeConnections(networkVals)

	time.Sleep(10 * time.Second)

	return networkExes, networkProps, networkVals
}

func createLedger(logs []tplog.Logger, rootDir string, backendType backend.BackendType, n int) []ledger.Ledger {
	if len(logs) != n {
		panic("Invalid logs, expected count: " + fmt.Sprintf("%d", n))
	}

	var ledgers []ledger.Ledger
	for i := 0; i < n; i++ {
		ledgerID := fmt.Sprintf("%d", i+1)
		ledgers = append(ledgers, ledger.NewLedger(rootDir, ledger.LedgerID(ledgerID), logs[i], backendType))
	}

	return ledgers
}

func TestMultiRoleNodes(t *testing.T) {
	tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

}
