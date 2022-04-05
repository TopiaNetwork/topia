package consensus

import (
	"context"
	"fmt"
	"github.com/TopiaNetwork/topia/chain"
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/consensus"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/integration/mock"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	"github.com/TopiaNetwork/topia/state"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
)

const (
	ExecutorNode_Number  = 3
	ProposerNode_Number  = 3
	ValidatorNode_number = 6
)

type nodeParams struct {
	nodeID    string
	nodeType  string
	priKey    tpcrtypes.PrivateKey
	mainLevel tplogcmm.LogLevel
	mainLog   tplog.Logger
	codecType codec.CodecType
	network   tpnet.Network
	txPool    txpool.TransactionPool
	ledger    ledger.Ledger
	config    *tpconfig.Configuration
	sysActor  *actor.ActorSystem
	compState state.CompositionState
}

func buildNodeConnections(networks []tpnet.Network) {
	if len(networks) <= 1 {
		return
	}

	for i := 1; i < len(networks); i++ {
		networks[i].Connect(networks[0].ListenAddr())
	}

	buildNodeConnections(networks[1:])
}

func createNetworkNodes(
	executorNetParams []*nodeParams,
	proposerNetParams []*nodeParams,
	validatorNetParams []*nodeParams,
	t *testing.T) ([]tpnet.Network, []tpnet.Network, []tpnet.Network) {
	var networkExes []tpnet.Network
	for i := 0; i < len(executorNetParams); i++ {
		network := tpnet.NewNetwork(context.Background(), executorNetParams[i].mainLog, executorNetParams[i].sysActor, fmt.Sprintf("/ip4/127.0.0.1/tcp/4100%d", i), "topia1", executorNetParams[i].compState)
		executorNetParams[i].mainLog.Infof("Execute network %d id=%s", i, network.ID())
		executorNetParams[i].nodeID = network.ID()
		executorNetParams[i].network = network
		networkExes = append(networkExes, network)

		executorNetParams[i].compState.AddNode(&chain.NodeInfo{
			NodeID: network.ID(),
			Weight: 10,
			Role:   chain.NodeRole_Executor,
			State:  chain.NodeState_Active,
		})
		for _, proposerNetParam := range proposerNetParams {
			proposerNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   chain.NodeRole_Proposer,
				State:  chain.NodeState_Active,
			})
		}
		for _, validatorNetParam := range validatorNetParams {
			validatorNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   chain.NodeRole_Validator,
				State:  chain.NodeState_Active,
			})
		}
	}

	var networkProps []tpnet.Network
	for i := 0; i < len(proposerNetParams); i++ {
		network := tpnet.NewNetwork(context.Background(), proposerNetParams[i].mainLog, proposerNetParams[i].sysActor, fmt.Sprintf("/ip4/127.0.0.1/tcp/5100%d", i), "topia1", proposerNetParams[i].compState)
		proposerNetParams[i].mainLog.Infof("Propose network %d id=%s", i, network.ID())
		proposerNetParams[i].nodeID = network.ID()
		proposerNetParams[i].network = network
		networkProps = append(networkProps, network)

		proposerNetParams[i].compState.AddNode(&chain.NodeInfo{
			NodeID: network.ID(),
			Weight: 10,
			Role:   chain.NodeRole_Proposer,
			State:  chain.NodeState_Active,
		})
		for _, executorNetParam := range executorNetParams {
			executorNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   chain.NodeRole_Executor,
				State:  chain.NodeState_Active,
			})
		}
		for _, validatorNetParam := range validatorNetParams {
			validatorNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   chain.NodeRole_Validator,
				State:  chain.NodeState_Active,
			})
		}
	}

	var networkVals []tpnet.Network
	for i := 0; i < len(validatorNetParams); i++ {
		network := tpnet.NewNetwork(context.Background(), validatorNetParams[i].mainLog, validatorNetParams[i].sysActor, fmt.Sprintf("/ip4/127.0.0.1/tcp/6100%d", i), "topia1", validatorNetParams[i].compState)
		validatorNetParams[i].mainLog.Infof("Validate network %d id=%s", i, network.ID())
		validatorNetParams[i].nodeID = network.ID()
		validatorNetParams[i].network = network
		networkVals = append(networkVals, network)

		validatorNetParams[i].compState.AddNode(&chain.NodeInfo{
			NodeID: network.ID(),
			Weight: 10,
			Role:   chain.NodeRole_Validator,
			State:  chain.NodeState_Active,
		})
		for _, executorNetParam := range executorNetParams {
			executorNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   chain.NodeRole_Executor,
				State:  chain.NodeState_Active,
			})
		}
		for _, proposerNetParam := range proposerNetParams {
			proposerNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   chain.NodeRole_Proposer,
				State:  chain.NodeState_Active,
			})
		}
	}

	buildNodeConnections(networkExes)
	buildNodeConnections(networkProps)
	buildNodeConnections(networkVals)

	time.Sleep(10 * time.Second)

	return networkExes, networkProps, networkVals
}

func createLedger(log tplog.Logger, rootDir string, backendType backend.BackendType, i int, nodeType string) ledger.Ledger {
	ledgerID := fmt.Sprintf("%s%d", nodeType, i+1)

	return ledger.NewLedger(rootDir, ledger.LedgerID(ledgerID), log, backendType)
}

func createNodeParams(n int, nodeType string) []*nodeParams {
	var nParams []*nodeParams

	for i := 0; i < n; i++ {
		testMainLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

		cryptService := &consensus.CryptServiceMock{}
		priKey, _, err := cryptService.GeneratePriPubKey()
		if err != nil {
			panic("Can't generate node private key")
		}

		txPool := &mock.TransactionPoolMock{}

		config := tpconfig.GetConfiguration()

		sysActor := actor.NewActorSystem()
		ledger := createLedger(testMainLog, "./TestConsensus", backend.BackendType_Badger, i, nodeType)
		compState := state.CreateCompositionState(testMainLog, ledger)

		nParams = append(nParams, &nodeParams{
			nodeType:  nodeType,
			priKey:    priKey,
			mainLevel: tplogcmm.InfoLevel,
			mainLog:   testMainLog,
			codecType: codec.CodecType_PROTO,
			txPool:    txPool,
			ledger:    ledger,
			config:    config,
			sysActor:  sysActor,
			compState: compState,
		})
	}

	return nParams
}

func createConsensusAndStart(nParams []*nodeParams) []consensus.Consensus {
	var css []consensus.Consensus
	for i := 0; i < len(nParams); i++ {
		cs := consensus.NewConsensus(
			nParams[i].nodeID,
			nParams[i].priKey,
			tplogcmm.InfoLevel,
			nParams[i].mainLog,
			codec.CodecType_PROTO,
			nParams[i].network,
			nParams[i].txPool,
			nParams[i].ledger,
			nParams[i].config.CSConfig,
		)

		cs.Start(nParams[i].sysActor)

		css = append(css, cs)
	}

	return css
}

func TestMultiRoleNodes(t *testing.T) {
	executorParams := createNodeParams(ExecutorNode_Number, "executor")
	proposerParams := createNodeParams(ExecutorNode_Number, "proposer")
	validatorParams := createNodeParams(ExecutorNode_Number, "validator")

	/*executorNet, proposerNet, validatorNet := */
	createNetworkNodes(executorParams, proposerParams, validatorParams, t)

	var nParams []*nodeParams
	nParams = append(nParams, executorParams...)
	nParams = append(nParams, proposerParams...)
	nParams = append(nParams, validatorParams...)

	createConsensusAndStart(nParams)

	time.Sleep(2000 * time.Millisecond)
}
