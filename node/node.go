package node

import (
	"context"
	"fmt"
	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/execution"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/consensus"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/state"
	"github.com/TopiaNetwork/topia/sync"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
)

type Node struct {
	log       tplog.Logger
	level     tplogcmm.LogLevel
	sysActor  *actor.ActorSystem
	marshaler codec.Marshaler
	evHub     eventhub.EventHub
	network   tpnet.Network
	ledger    ledger.Ledger
	consensus consensus.Consensus
	txPool    txpool.TransactionPool
	syncer    sync.Syncer
	chain     chain.Chain
	config    *tpconfig.Configuration
}

func NewNode(endPoint string, seed string) *Node {
	homeDir, _ := os.UserHomeDir()
	chainRootPath := filepath.Join(homeDir, "topia")

	mainLog, err := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")
	if err != nil {
		fmt.Printf("CreateMainLogger error: %v", err)
	}

	ctx := context.Background()

	sysActor := actor.NewActorSystem()

	var priKey tpcrtypes.PrivateKey

	config := tpconfig.GetConfiguration()

	ledger := ledger.NewLedger(chainRootPath, "topia", mainLog, backend.BackendType_Badger)

	compStateRN := state.CreateCompositionStateReadonly(mainLog, ledger)
	defer compStateRN.Stop()

	network := tpnet.NewNetwork(ctx, mainLog, sysActor, endPoint, seed, state.NewNodeNetWorkStateWapper(mainLog, ledger))
	nodeID := network.ID()
	txPool := txpool.NewTransactionPool(tplogcmm.InfoLevel, mainLog, codec.CodecType_PROTO)
	exeScheduler := execution.NewExecutionScheduler(nodeID, mainLog, config, txPool)
	evHub := eventhub.GetEventHubManager().CreateEventHub(nodeID, tplogcmm.InfoLevel, mainLog)
	cons := consensus.NewConsensus(compStateRN.ChainID(), nodeID, priKey, tplogcmm.InfoLevel, mainLog, codec.CodecType_PROTO, network, txPool, ledger, exeScheduler, config)
	syncer := sync.NewSyncer(tplogcmm.InfoLevel, mainLog, codec.CodecType_PROTO)
	chain := chain.NewChain(tplogcmm.InfoLevel, mainLog, nodeID, codec.CodecType_PROTO, ledger, exeScheduler, config)

	return &Node{
		log:       mainLog,
		level:     tplogcmm.InfoLevel,
		sysActor:  sysActor,
		marshaler: codec.CreateMarshaler(codec.CodecType_PROTO),
		evHub:     evHub,
		network:   network,
		ledger:    ledger,
		consensus: cons,
		txPool:    txPool,
		syncer:    syncer,
		chain:     chain,
		config:    config,
	}
}

func (n *Node) Start() {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	var waitChannel = make(chan bool)

	go func() {
		sig := <-gracefulStop
		n.log.Debugf("caught sig: ", sig)

		n.log.Warn("GRACEFUL STOP APP")
		n.log.Info("main leave ends ")

		n.Stop()

		close(waitChannel)
	}()

	var err error
	var latestEpochInfo *tpcmm.EpochInfo
	var latestBlock *tpchaintypes.Block
	if n.ledger.State() == tpcmm.LedgerState_Uninitialized {
		compState := state.GetStateBuilder().CompositionState(n.network.ID(), 1)
		err = compState.SetLatestEpoch(n.config.Genesis.Epon)
		if err != nil {
			n.log.Panicf("Set latest epoch of genesis error: %v", err)
			compState.Stop()
			return
		}
		err = compState.SetLatestBlock(n.config.Genesis.Block)
		if err != nil {
			n.log.Panicf("Set latest block of genesis error: %v", err)
			compState.Stop()
			return
		}

		err = compState.SetLatestBlockResult(n.config.Genesis.BlockResult)
		if err != nil {
			n.log.Panicf("Set latest block result of genesis error: %v", err)
			compState.Stop()
			return
		}

		compState.Commit()

		latestEpochInfo = n.config.Genesis.Epon
		latestBlock = n.config.Genesis.Block
	} else {
		csStateRN := state.CreateCompositionStateReadonly(n.log, n.ledger)
		defer csStateRN.Stop()

		latestEpochInfo, err = csStateRN.GetLatestEpoch()
		if err != nil {
			n.log.Panicf("Can't get the latest epoch info error: %v", err)
			return
		}

		latestBlock, err = csStateRN.GetLatestBlock()
		if err != nil {
			n.log.Panicf("Can't get he latest block info error: %v", err)
			return
		}
	}

	n.evHub.Start(n.sysActor)
	n.network.Start()
	n.consensus.Start(n.sysActor, latestEpochInfo.Epoch, latestEpochInfo.StartTimeStamp, latestBlock.Head.Height)
	n.txPool.Start(n.sysActor, n.network)
	n.syncer.Start(n.sysActor, n.network)
	n.chain.Start(n.sysActor, n.network)

	fmt.Println("All services were started")
	<-waitChannel
}

func (n *Node) Stop() {
	n.consensus.Stop()
	n.syncer.Stop()
	n.network.Stop()
}
