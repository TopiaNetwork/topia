package consensus

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"testing"
	"time"

	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/util/encoding"
	"github.com/TopiaNetwork/kyber/v3/util/key"

	"github.com/AsynkronIT/protoactor-go/actor"
	actorlog "github.com/AsynkronIT/protoactor-go/log"
	tpacc "github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/consensus"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/integration/mock"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/state"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

const (
	ExecutorNode_Number  = 6
	ProposerNode_Number  = 3
	ValidatorNode_number = 4
	archiverNode_number  = 1
)

var portFrefix = map[string]string{
	"executor":  "4100",
	"proposer":  "5100",
	"validator": "6100",
}

type nodeParams struct {
	chainID         tpchaintypes.ChainID
	nodeID          string
	nodeType        string
	priKey          tpcrtypes.PrivateKey
	dkgPriKey       string
	dkgPartPubKey   string
	mainLevel       tplogcmm.LogLevel
	mainLog         tplog.Logger
	codecType       codec.CodecType
	network         tpnet.Network
	txPool          txpooli.TransactionPool
	ledger          ledger.Ledger
	scheduler       execution.ExecutionScheduler
	cs              consensus.Consensus
	chain           chain.Chain
	config          *tpconfig.Configuration
	sysActor        *actor.ActorSystem
	compState       state.CompositionState
	latestEpochInfo *tpcmm.EpochInfo
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
	archiverNetParams []*nodeParams,
	t *testing.T) ([]tpnet.Network, []tpnet.Network, []tpnet.Network) {
	var networkExes []tpnet.Network
	suite := bn256.NewSuiteG2()

	exeDomainInfo1 := &tpcmm.NodeDomainInfo{
		ID:               tpcmm.CreateDomainID("exedomain1"),
		Type:             tpcmm.DomainType_Execute,
		ValidHeightStart: 1,
		ValidHeightEnd:   100000,
		ExeDomainData:    new(tpcmm.NodeExecuteDomain),
	}

	exeDomainInfo2 := &tpcmm.NodeDomainInfo{
		ID:               tpcmm.CreateDomainID("exedomain2"),
		Type:             tpcmm.DomainType_Execute,
		ValidHeightStart: 1,
		ValidHeightEnd:   100000,
		ExeDomainData:    new(tpcmm.NodeExecuteDomain),
	}

	for i := 0; i < len(executorNetParams); i++ {
		if i < len(executorNetParams)/2 {
			exeDomainInfo1.ExeDomainData.Members = append(exeDomainInfo1.ExeDomainData.Members, executorNetParams[i].network.ID())
		} else {
			exeDomainInfo2.ExeDomainData.Members = append(exeDomainInfo2.ExeDomainData.Members, executorNetParams[i].network.ID())
		}
	}

	for i := 0; i < len(executorNetParams); i++ {
		network := executorNetParams[i].network
		executorNetParams[i].mainLog.Infof("Execute network %d id=%s", i, network.ID())
		executorNetParams[i].nodeID = network.ID()
		networkExes = append(networkExes, network)

		executorNetParams[i].compState.AddNode(&tpcmm.NodeInfo{
			NodeID: network.ID(),
			Weight: 10,
			Role:   tpcmm.NodeRole_Executor,
			State:  tpcmm.NodeState_Active,
		})

		if i < len(executorNetParams)/2 {
			executorNetParams[i].compState.AddNodeDomain(exeDomainInfo1)
		} else {
			executorNetParams[i].compState.AddNodeDomain(exeDomainInfo2)
		}

		for j := 0; j < i; j++ {
			executorNetParams[j].compState.AddNode(&tpcmm.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   tpcmm.NodeRole_Executor,
				State:  tpcmm.NodeState_Active,
			})
			executorNetParams[i].compState.AddNode(&tpcmm.NodeInfo{
				NodeID: executorNetParams[j].nodeID,
				Weight: 10,
				Role:   tpcmm.NodeRole_Executor,
				State:  tpcmm.NodeState_Active,
			})
		}
		for _, proposerNetParam := range proposerNetParams {
			proposerNetParam.compState.AddNode(&tpcmm.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   tpcmm.NodeRole_Executor,
				State:  tpcmm.NodeState_Active,
			})
		}
		for _, validatorNetParam := range validatorNetParams {
			validatorNetParam.compState.AddNode(&tpcmm.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   tpcmm.NodeRole_Executor,
				State:  tpcmm.NodeState_Active,
			})
		}
		for _, archiverNetParam := range archiverNetParams {
			archiverNetParam.compState.AddNode(&tpcmm.NodeInfo{
				NodeID: network.ID(),
				Weight: 10,
				Role:   tpcmm.NodeRole_Executor,
				State:  tpcmm.NodeState_Active,
			})
		}
	}

	for i, archiverNetParam := range archiverNetParams {
		archiverNetParam.mainLog.Infof("Archiver network %d id=%s", i, archiverNetParam.network.ID())
		archiverNetParam.compState.AddNodeDomain(exeDomainInfo1)
		archiverNetParam.compState.AddNodeDomain(exeDomainInfo2)
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
		proposerNetParams[i].config.CSConfig.InitDKGPrivKey = proposerNetParams[i].dkgPriKey

		proposerNetParams[i].compState.AddNode(&tpcmm.NodeInfo{
			NodeID:        network.ID(),
			Weight:        uint64(10 * (i + 1)),
			DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
			Role:          tpcmm.NodeRole_Proposer,
			State:         tpcmm.NodeState_Active,
		})

		proposerNetParams[i].compState.AddNodeDomain(exeDomainInfo1)
		proposerNetParams[i].compState.AddNodeDomain(exeDomainInfo2)

		for j := 0; j < i; j++ {
			proposerNetParams[j].compState.AddNode(&tpcmm.NodeInfo{
				NodeID:        network.ID(),
				Weight:        uint64(10 * (i + 1)),
				DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
				Role:          tpcmm.NodeRole_Proposer,
				State:         tpcmm.NodeState_Active,
			})

			proposerNetParams[i].compState.AddNode(&tpcmm.NodeInfo{
				NodeID:        proposerNetParams[j].nodeID,
				Weight:        uint64(10 * (i + 1)),
				DKGPartPubKey: proposerNetParams[j].dkgPartPubKey,
				Role:          tpcmm.NodeRole_Proposer,
				State:         tpcmm.NodeState_Active,
			})
		}
		for _, executorNetParam := range executorNetParams {
			executorNetParam.compState.AddNode(&tpcmm.NodeInfo{
				NodeID:        network.ID(),
				Weight:        uint64(10 * (i + 1)),
				DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
				Role:          tpcmm.NodeRole_Proposer,
				State:         tpcmm.NodeState_Active,
			})
		}
		for _, validatorNetParam := range validatorNetParams {
			validatorNetParam.compState.AddNode(&tpcmm.NodeInfo{
				NodeID:        network.ID(),
				Weight:        uint64(10 * (i + 1)),
				DKGPartPubKey: proposerNetParams[i].dkgPartPubKey,
				Role:          tpcmm.NodeRole_Proposer,
				State:         tpcmm.NodeState_Active,
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
		validatorNetParams[i].dkgPriKey, _ = encoding.ScalarToStringHex(suite, keyPair.Private)
		validatorNetParams[i].dkgPartPubKey, _ = encoding.PointToStringHex(suite, keyPair.Public)
		validatorNetParams[i].config.CSConfig.InitDKGPrivKey = validatorNetParams[i].dkgPriKey

		validatorNetParams[i].compState.AddNode(&tpcmm.NodeInfo{
			NodeID:        network.ID(),
			Weight:        10,
			DKGPartPubKey: validatorNetParams[i].dkgPartPubKey,
			Role:          tpcmm.NodeRole_Validator,
			State:         tpcmm.NodeState_Active,
		})
		validatorNetParams[i].compState.AddNodeDomain(exeDomainInfo1)
		validatorNetParams[i].compState.AddNodeDomain(exeDomainInfo2)
		for j := 0; j < i; j++ {
			validatorNetParams[j].compState.AddNode(&tpcmm.NodeInfo{
				NodeID:        network.ID(),
				Weight:        10,
				DKGPartPubKey: validatorNetParams[i].dkgPartPubKey,
				Role:          tpcmm.NodeRole_Validator,
				State:         tpcmm.NodeState_Active,
			})
			validatorNetParams[i].compState.AddNode(&tpcmm.NodeInfo{
				NodeID:        validatorNetParams[j].nodeID,
				Weight:        10,
				DKGPartPubKey: validatorNetParams[j].dkgPartPubKey,
				Role:          tpcmm.NodeRole_Validator,
				State:         tpcmm.NodeState_Active,
			})
		}
		for _, executorNetParam := range executorNetParams {
			executorNetParam.compState.AddNode(&tpcmm.NodeInfo{
				NodeID:        network.ID(),
				Weight:        10,
				DKGPartPubKey: validatorNetParams[i].dkgPartPubKey,
				Role:          tpcmm.NodeRole_Validator,
				State:         tpcmm.NodeState_Active,
			})
		}
		for _, proposerNetParam := range proposerNetParams {
			proposerNetParam.compState.AddNode(&tpcmm.NodeInfo{
				NodeID:        network.ID(),
				Weight:        10,
				DKGPartPubKey: validatorNetParams[i].dkgPartPubKey,
				Role:          tpcmm.NodeRole_Validator,
				State:         tpcmm.NodeState_Active,
			})
		}
	}

	//buildNodeConnections(networkExes)
	//buildNodeConnections(networkProps)
	//buildNodeConnections(networkVals)

	var netCons []tpnet.Network
	var netARCons []tpnet.Network
	for i, netExe := range networkExes {
		netCons = append(netCons, netExe)
		if i < 2 {
			netARCons = append(netARCons, netExe)
		}
	}
	for _, netProp := range networkProps {
		netCons = append(netCons, netProp)
	}
	for _, netVal := range networkVals {
		netCons = append(netCons, netVal)
	}

	buildNodeConnections(netCons)

	for _, netAr := range archiverNetParams {
		for _, target := range netARCons {
			err := netAr.network.Connect(target.ListenAddr())
			if err != nil {
				netAr.mainLog.Panicf("err: %v")
			}
		}
	}

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

		config := tpconfig.GetConfiguration()

		sysActor := actor.NewActorSystem()

		actor.SetLogLevel(actorlog.DebugLevel)

		l := createLedger(testMainLog, "./TestConsensus", backend.BackendType_Badger, i, nodeType)

		network := tpnet.NewNetwork(context.Background(), testMainLog, config.NetConfig, sysActor, fmt.Sprintf("/ip4/127.0.0.1/tcp/%s%d", portFrefix[nodeType], i), fmt.Sprintf("topia%s%d", portFrefix[nodeType], i+1), state.NewNodeNetWorkStateWapper(testMainLog, l))

		txPool := mock.NewTransactionPoolMock(testMainLog, network.ID(), cryptService)

		eventhub.GetEventHubManager().CreateEventHub(network.ID(), tplogcmm.InfoLevel, testMainLog)

		exeScheduler := execution.NewExecutionScheduler(network.ID(), testMainLog, config, codec.CodecType_PROTO, txPool)

		chain := chain.NewChain(tplogcmm.InfoLevel, testMainLog, network.ID(), codec.CodecType_PROTO, l, txPool, exeScheduler, config)

		cType := state.CompStateBuilderType_Full
		if nodeType != "executor" {
			cType = state.CompStateBuilderType_Simple
			for i := 1; i <= 100; i++ {
				state.GetStateBuilder(cType).CreateCompositionState(testMainLog, network.ID(), l, uint64(i), "tester")
			}
		}
		compState := state.GetStateBuilder(cType).CreateCompositionState(testMainLog, network.ID(), l, 1, "tester")

		var latestEpochInfo *tpcmm.EpochInfo
		var latestBlock *tpchaintypes.Block
		if l.State() == tpcmm.LedgerState_Uninitialized {
			err = compState.SetLatestEpoch(config.Genesis.Epoch)
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

			latestEpochInfo = config.Genesis.Epoch
			latestBlock = config.Genesis.Block

			compState.AddAccount(tpacc.NativeContractAccount_Account)

			l.UpdateState(tpcmm.LedgerState_Genesis)
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

		newConfig := &tpconfig.Configuration{}
		tpcmm.DeepCopyByGob(newConfig, config)
		nParams = append(nParams, &nodeParams{
			chainID:         "consintetest",
			nodeType:        nodeType,
			priKey:          priKey,
			mainLevel:       tplogcmm.InfoLevel,
			mainLog:         testMainLog,
			codecType:       codec.CodecType_PROTO,
			network:         network,
			txPool:          txPool,
			ledger:          l,
			scheduler:       exeScheduler,
			chain:           chain,
			config:          newConfig,
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
		eventhub.GetEventHubManager().GetEventHub(nParams[i].nodeID).Start(nParams[i].sysActor)

		if nParams[i].nodeType == "archiver" {
			nParams[i].chain.Start(nParams[i].sysActor, nParams[i].network)
			continue
		}

		cType := state.CompStateBuilderType_Full
		if nParams[i].nodeType != "executor" {
			cType = state.CompStateBuilderType_Simple
		}
		cs := consensus.NewConsensus(
			nParams[i].chainID,
			nParams[i].nodeID,
			cType,
			nParams[i].priKey,
			tplogcmm.InfoLevel,
			nParams[i].mainLog,
			codec.CodecType_PROTO,
			nParams[i].network,
			nParams[i].txPool,
			nParams[i].ledger,
			nParams[i].scheduler,
			nParams[i].config,
		)

		cs.Start(nParams[i].sysActor, nParams[i].latestEpochInfo.Epoch, nParams[i].latestEpochInfo.StartHeight, nParams[i].latestBlock.Head.Height)
		nParams[i].cs = cs
		css = append(css, cs)

		if nParams[i].nodeType == "executor" {
			nParams[i].chain.Start(nParams[i].sysActor, nParams[i].network)
			nParams[i].txPool.Start(nParams[i].sysActor, nParams[i].network)
		}
	}

	return css
}

func TestMultiRoleNodes(t *testing.T) {
	os.RemoveAll("./TestConsensus")
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	waitChan := make(chan struct{})

	executorParams := createNodeParams(ExecutorNode_Number, "executor")
	proposerParams := createNodeParams(ProposerNode_Number, "proposer")
	validatorParams := createNodeParams(ValidatorNode_number, "validator")
	archiverParams := createNodeParams(archiverNode_number, "archiver")

	/*executorNet, proposerNet, validatorNet := */
	createNetworkNodes(executorParams, proposerParams, validatorParams, archiverParams, t)

	var nParams []*nodeParams
	nParams = append(nParams, executorParams...)
	nParams = append(nParams, proposerParams...)
	nParams = append(nParams, validatorParams...)
	for i, nodeP := range nParams {
		nodeP.compState.Commit()
		nodeP.compState.UpdateCompSState(state.CompSState_Commited)
		ppPort := 8899 + i
		go func() {
			http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", ppPort), nil)
		}()
		t.Logf("DKG PriKey = %s, id=%s, nodeType=%s, ppPort=%d", nodeP.config.CSConfig.InitDKGPrivKey, nodeP.nodeID, nodeP.nodeType, ppPort)
	}

	time.Sleep(10 * time.Second)

	createConsensusAndStart(nParams)

	var dkgNParams []*nodeParams
	dkgNParams = append(dkgNParams, proposerParams...)
	dkgNParams = append(nParams, validatorParams...)
	for _, nodeP := range dkgNParams {
		func() {
			csStateRN := state.CreateCompositionStateReadonly(nodeP.mainLog, nodeP.ledger)
			defer csStateRN.Stop()

			activeProposers, _ := csStateRN.GetActiveProposerIDs()
			activeValidators, _ := csStateRN.GetActiveValidatorIDs()

			nodeP.mainLog.Infof("Current active proposers %v and validators %v", activeProposers, activeValidators)

			latestBlock, _ := csStateRN.GetLatestBlock()

			if nodeP.nodeType != "executor" && nodeP.ledger.State() == tpcmm.LedgerState_Genesis {
				nodeP.cs.TriggerDKG(latestBlock)
			}
		}()
	}

	<-waitChan
	//time.Sleep(10000 * time.Millisecond)
}
