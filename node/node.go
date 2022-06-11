package node

import (
	"context"
	"encoding/hex"
	"fmt"
	tpacc "github.com/TopiaNetwork/topia/account"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/consensus"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/service"
	"github.com/TopiaNetwork/topia/state"
	"github.com/TopiaNetwork/topia/sync"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
	"github.com/TopiaNetwork/topia/wallet"
)

type Node struct {
	log         tplog.Logger
	level       tplogcmm.LogLevel
	sysActor    *actor.ActorSystem
	chainID     tpchaintypes.ChainID
	networkType tpcmm.NetworkType
	role        string
	latestEpoch *tpcmm.EpochInfo
	latestBlock *tpchaintypes.Block
	marshaler   codec.Marshaler
	evHub       eventhub.EventHub
	network     tpnet.Network
	ledger      ledger.Ledger
	consensus   consensus.Consensus
	txPool      txpooli.TransactionPool
	syncer      sync.Syncer
	chain       chain.Chain
	config      *tpconfig.Configuration
	service     service.Service
}

func NewNode(rootPath string, endPoint string, seed string, role string) *Node {
	if rootPath == "" {
		rootPath, _ = os.UserHomeDir()
	}
	chainRootPath := filepath.Join(rootPath, "topia")

	ctx := context.Background()

	n := &Node{}

	n.role = role

	n.marshaler = codec.CreateMarshaler(codec.CodecType_PROTO)

	mainLog, err := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")
	if err != nil {
		fmt.Printf("CreateMainLogger error: %v", err)
	}
	n.level = tplogcmm.InfoLevel
	n.log = mainLog

	n.sysActor = actor.NewActorSystem()

	priKeyBytes, _ := hex.DecodeString(tpconfig.TestDatas[seed].PrivKey)
	priKey := tpcrtypes.PrivateKey(priKeyBytes)

	config := tpconfig.GetConfiguration()
	config.CSConfig.InitDKGPrivKey = tpconfig.TestDatas[seed].InitDKGPrivKey
	n.config = config

	n.ledger = ledger.NewLedger(chainRootPath, "topia", mainLog, backend.BackendType_Badger)

	n.network = tpnet.NewNetwork(ctx, mainLog, config.NetConfig, n.sysActor, endPoint, seed, state.NewNodeNetWorkStateWapper(mainLog, n.ledger))

	n.initData()

	nodeID := n.network.ID()

	w := wallet.NewWallet(tplogcmm.InfoLevel, mainLog, chainRootPath)
	service := service.NewService(nodeID, mainLog, codec.CodecType_PROTO, n.network, n.ledger, nil, w, config)
	txPoolConf := txpooli.DefaultTransactionPoolConfig
	n.txPool = txpool.NewTransactionPool(nodeID, ctx, txPoolConf, tplogcmm.InfoLevel, mainLog, codec.CodecType_PROTO, service.StateQueryService(), service.BlockService(), n.network)

	service.SetTxPool(n.txPool)
	n.service = service

	exeScheduler := execution.NewExecutionScheduler(nodeID, mainLog, config, codec.CodecType_PROTO, n.txPool)
	n.chain = chain.NewChain(tplogcmm.InfoLevel, mainLog, nodeID, codec.CodecType_PROTO, n.ledger, n.txPool, exeScheduler, config)

	n.evHub = eventhub.GetEventHubManager().CreateEventHub(nodeID, tplogcmm.InfoLevel, mainLog)

	n.consensus = consensus.NewConsensus(n.chainID, nodeID, priKey, tplogcmm.InfoLevel, mainLog, codec.CodecType_PROTO, n.network, n.txPool, n.ledger, exeScheduler, config)

	n.syncer = sync.NewSyncer(tplogcmm.InfoLevel, mainLog, codec.CodecType_PROTO)

	return n
}

func (n *Node) initData() {
	var err error
	var chainID tpchaintypes.ChainID
	var netType tpcmm.NetworkType
	var latestEpochInfo *tpcmm.EpochInfo
	var latestBlock *tpchaintypes.Block
	if n.ledger.State() == tpcmm.LedgerState_Uninitialized {
		cType := state.CompStateBuilderType_Full
		if n.role != "executor" {
			cType = state.CompStateBuilderType_Simple
			for i := 1; i <= 100; i++ {
				state.GetStateBuilder(cType).CreateCompositionState(n.log, n.network.ID(), n.ledger, uint64(i), "node")
			}
		}
		compState := state.GetStateBuilder(cType).CreateCompositionState(n.log, n.network.ID(), n.ledger, 1, "node")

		err = compState.SetLatestEpoch(n.config.Genesis.Epoch)
		if err != nil {
			n.log.Panicf("Set latest epoch of genesis error: %v", err)
			compState.Stop()
			return
		}

		err = compState.SetChainID(n.config.Genesis.ChainID)
		if err != nil {
			n.log.Panicf("Set chain id of genesis error: %v", err)
			compState.Stop()
			return
		}

		err = compState.SetNetworkType(n.config.Genesis.NetType)
		if err != nil {
			n.log.Panicf("Set network type of genesis error: %v", err)
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

		for _, nodeInfo := range n.config.Genesis.GenesisNode {
			err = compState.AddNode(nodeInfo)
			if err != nil {
				n.log.Panicf("Add node info error: %v", err)
				compState.Stop()
				return
			}
		}

		compState.AddAccount(tpacc.NativeContractAccount_Account)

		n.ledger.UpdateState(tpcmm.LedgerState_Genesis)

		compState.Commit()
		compState.UpdateCompSState(state.CompSState_Commited)

		chainID = n.config.Genesis.ChainID
		netType = n.config.Genesis.NetType
		latestEpochInfo = n.config.Genesis.Epoch
		latestBlock = n.config.Genesis.Block
	} else {
		csStateRN := state.CreateCompositionStateReadonly(n.log, n.ledger)
		defer csStateRN.Stop()

		chainID = csStateRN.ChainID()

		netType = csStateRN.NetworkType()

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

	n.chainID = chainID
	n.networkType = netType
	n.latestEpoch = latestEpochInfo
	n.latestBlock = latestBlock
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

	n.evHub.Start(n.sysActor)
	n.network.Start()
	n.consensus.Start(n.sysActor, n.latestEpoch.Epoch, n.latestEpoch.StartTimeStamp, n.latestBlock.Head.Height)
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
