package service

import (
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
	"github.com/TopiaNetwork/topia/wallet"
)

type Service interface {
	SetTxPool(txPool txpooli.TransactionPool)

	StateQueryService() StateQueryService

	StateQueryServiceAt(version uint64) StateQueryService

	NetworkService() NetworkService

	BlockService() BlockService

	TransactionService() TransactionService

	WalletService() WalletService

	ContractService() ContractService

	AccountService() AccountService

	TxPoolService() TxPoolService

	SyncService() SyncService
}

type service struct {
	nodeID    string
	log       tplog.Logger
	marshaler codec.Marshaler
	network   tpnet.Network
	ledger    ledger.Ledger
	txPool    txpooli.TransactionPool
	w         wallet.Wallet
	config    *tpconfig.Configuration
}

func NewService(nodeID string,
	log tplog.Logger,
	codecType codec.CodecType,
	network tpnet.Network,
	ledger ledger.Ledger,
	txPool txpooli.TransactionPool,
	w wallet.Wallet,
	config *tpconfig.Configuration) Service {
	return &service{
		nodeID:    nodeID,
		log:       log,
		marshaler: codec.CreateMarshaler(codecType),
		network:   network,
		ledger:    ledger,
		txPool:    txPool,
		w:         w,
		config:    config,
	}
}

func (s *service) SetTxPool(txPool txpooli.TransactionPool) {
	s.txPool = txPool
}

func (s *service) StateQueryService() StateQueryService {
	var sqProxyObj stateQueryProxyObject
	stateQueryProxy(s.log, s.ledger, &sqProxyObj, nil)
	return &sqProxyObj
}

func (s *service) StateQueryServiceAt(version uint64) StateQueryService {
	var sqProxyObj stateQueryProxyObject
	stateQueryProxy(s.log, s.ledger, &sqProxyObj, &version)
	return &sqProxyObj
}

func (s *service) NetworkService() NetworkService {
	return NewNetworkService(s.network)
}

func (s *service) BlockService() BlockService {
	return &blockService{s.ledger.GetBlockStore()}
}

func (s *service) TransactionService() TransactionService {
	return newTransactionService(s.nodeID, s.log, s.marshaler, s.network, s.ledger, s.txPool, s.StateQueryService(), s.config)
}

func (s *service) WalletService() WalletService {
	return NewWalletService(s.w)
}

func (s *service) ContractService() ContractService {
	return NewContractService(s.log, s.marshaler, s.StateQueryService(), s.TransactionService(), s.WalletService())
}

func (s *service) AccountService() AccountService {
	return NewAccountService(s.log, s.ContractService())
}

func (s *service) TxPoolService() TxPoolService {
	return NewTxPoolService(s.txPool)
}

func (s *service) SyncService() SyncService {
	return NewSyncService()
}
