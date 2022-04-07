package consensus

import (
	"context"
	"fmt"
	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/util/encoding"
	"github.com/TopiaNetwork/kyber/v3/util/key"
	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/consensus"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/integration/mock"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	"github.com/TopiaNetwork/topia/state"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
	"os"
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

var portFrefix = map[string]string{
	"executor":  "4100",
	"proposer":  "5100",
	"validator": "6100",
}

type nodeParams struct {
	nodeID          string
	nodeType        string
	priKey          tpcrtypes.PrivateKey
	dkgPriKey       string
	dkgPartPubKey   string
	mainLevel       tplogcmm.LogLevel
	mainLog         tplog.Logger
	codecType       codec.CodecType
	network         tpnet.Network
	txPool          txpool.TransactionPool
	ledger          ledger.Ledger
	config          *tpconfig.Configuration
	sysActor        *actor.ActorSystem
	compState       state.CompositionState
	latestEpochInfo *chain.EpochInfo
	latestBlock     *tpchaintypes.Block
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
	suite := bn256.NewSuiteG2()
	for i := 0; i < len(executorNetParams); i++ {
		network := executorNetParams[i].network
		executorNetParams[i].mainLog.Infof("Execute network %d id=%s", i, network.ID())
		executorNetParams[i].nodeID = network.ID()
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
				Role:   chain.NodeRole_Executor,
				State:  chain.NodeState_Active,
			})
		}
		for _, validatorNetParam := range validatorNetParams {
			validatorNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   chain.NodeRole_Executor,
				State:  chain.NodeState_Active,
			})
		}
	}

	var networkProps []tpnet.Network
	for i := 0; i < len(proposerNetParams); i++ {
		network := proposerNetParams[i].network
		proposerNetParams[i].mainLog.Infof("Propose network %d id=%s", i, network.ID())
		proposerNetParams[i].nodeID = network.ID()
		networkProps = append(networkProps, network)

		keyPair := key.NewKeyPair(suite)
		proposerNetParams[i].dkgPriKey, _ = encoding.ScalarToStringHex(suite, keyPair.Private)
		proposerNetParams[i].dkgPartPubKey, _ = encoding.PointToStringHex(suite, keyPair.Public)

		proposerNetParams[i].compState.AddNode(&chain.NodeInfo{
			NodeID:        network.ID(),
			Weight:        10,
			DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
			Role:          chain.NodeRole_Proposer,
			State:         chain.NodeState_Active,
		})
		for _, executorNetParam := range executorNetParams {
			executorNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID:        network.ID(),
				Weight:        10,
				DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
				Role:          chain.NodeRole_Proposer,
				State:         chain.NodeState_Active,
			})
		}
		for _, validatorNetParam := range validatorNetParams {
			validatorNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID:        network.ID(),
				Weight:        10,
				DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
				Role:          chain.NodeRole_Proposer,
				State:         chain.NodeState_Active,
			})
		}
	}

	var networkVals []tpnet.Network
	for i := 0; i < len(validatorNetParams); i++ {
		network := validatorNetParams[i].network
		validatorNetParams[i].mainLog.Infof("Validate network %d id=%s", i, network.ID())
		validatorNetParams[i].nodeID = network.ID()
		networkVals = append(networkVals, network)

		keyPair := key.NewKeyPair(suite)
		proposerNetParams[i].dkgPriKey, _ = encoding.ScalarToStringHex(suite, keyPair.Private)
		proposerNetParams[i].dkgPartPubKey, _ = encoding.PointToStringHex(suite, keyPair.Public)

		validatorNetParams[i].compState.AddNode(&chain.NodeInfo{
			NodeID: network.ID(),
			Weight: 10,
			Role:   chain.NodeRole_Validator,
			State:  chain.NodeState_Active,
		})
		for _, executorNetParam := range executorNetParams {
			executorNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID:        network.ID(),
				Weight:        10,
				DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
				Role:          chain.NodeRole_Validator,
				State:         chain.NodeState_Active,
			})
		}
		for _, proposerNetParam := range proposerNetParams {
			proposerNetParam.compState.AddNode(&chain.NodeInfo{
				NodeID:        network.ID(),
				Weight:        10,
				DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
				Role:          chain.NodeRole_Validator,
				State:         chain.NodeState_Active,
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

		network := tpnet.NewNetwork(context.Background(), testMainLog, sysActor, fmt.Sprintf("/ip4/127.0.0.1/tcp/%s%d", portFrefix[nodeType], i), fmt.Sprintf("topia%s%d", portFrefix[nodeType], i+1), state.NewNodeNetWorkStateWapper(testMainLog, ledger))

		eventhub.GetEventHub(network.ID(), tplogcmm.InfoLevel, testMainLog)

		compState := state.GetStateBuilder().CreateCompositionState(testMainLog, network.ID(), ledger, 1)

		var latestEpochInfo *chain.EpochInfo
		var latestBlock *tpchaintypes.Block
		if ledger.IsGenesisState() {
			err = compState.SetLatestEpoch(config.Genesis.Epon)
			if err != nil {
				panic("Set latest epoch of genesis error: " + err.Error())
				compState.Stop()
				return nil
			}
			err = compState.SetLatestBlock(config.Genesis.Block)
			if err != nil {
				panic("Set latest block of genesis error: " + err.Error())
				compState.Stop()
				return nil
			}

			err = compState.SetLatestBlockResult(config.Genesis.BlockResult)
			if err != nil {
				panic("Set latest block result of genesis error: " + err.Error())
				compState.Stop()
				return nil
			}

			latestEpochInfo = config.Genesis.Epon
			latestBlock = config.Genesis.Block
		} /*else {
			csStateRN := state.CreateCompositionStateReadonly(testMainLog, ledger)
			defer csStateRN.Stop()

			latestEpochInfo, err = csStateRN.GetLatestEpoch()
			if err != nil {
				panic("Can't get the latest epoch info error: " + err.Error())
			}

			latestBlock, err = csStateRN.GetLatestBlock()
			if err != nil {
				panic("Can't get he latest block info error: " + err.Error())
			}
		}*/

		nParams = append(nParams, &nodeParams{
			nodeType:        nodeType,
			priKey:          priKey,
			mainLevel:       tplogcmm.InfoLevel,
			mainLog:         testMainLog,
			codecType:       codec.CodecType_PROTO,
			network:         network,
			txPool:          txPool,
			ledger:          ledger,
			config:          config,
			sysActor:        sysActor,
			compState:       compState,
			latestEpochInfo: latestEpochInfo,
			latestBlock:     latestBlock,
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

		cs.Start(nParams[i].sysActor, nParams[i].latestEpochInfo.Epoch, nParams[i].latestEpochInfo.StartHeight, nParams[i].latestBlock.Head.Height)

		css = append(css, cs)
	}

	return css
}

func TestMultiRoleNodes(t *testing.T) {
	os.RemoveAll("./TestConsensus")

	executorParams := createNodeParams(ExecutorNode_Number, "executor")
	proposerParams := createNodeParams(ExecutorNode_Number, "proposer")
	validatorParams := createNodeParams(ExecutorNode_Number, "validator")

	/*executorNet, proposerNet, validatorNet := */
	createNetworkNodes(executorParams, proposerParams, validatorParams, t)

	var nParams []*nodeParams
	nParams = append(nParams, executorParams...)
	nParams = append(nParams, proposerParams...)
	nParams = append(nParams, validatorParams...)
	for _, nodeP := range nParams {
		nodeP.compState.Commit()
	}

	createConsensusAndStart(nParams)

	time.Sleep(2000 * time.Millisecond)
}
