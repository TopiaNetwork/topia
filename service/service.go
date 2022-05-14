package service

import (
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
	"github.com/TopiaNetwork/topia/wallet"
)

type Service interface {
}

type service struct {
	nodeID    string
	log       tplog.Logger
	marshaler codec.Marshaler
	network   tpnet.Network
	ledger    ledger.Ledger
	txPool    txpool.TransactionPool
	w         wallet.Wallet
}

func (s *service) StateQueryService() StateQueryService {
	var sqProxyObj stateQueryProxyObject
	stateQueryProxy(s.log, s.ledger, &sqProxyObj)
	return &sqProxyObj
}

func (s *service) NetworkService() NetworkService {
	return NewNetworkService(s.network)
}

func (s *service) BlockService() BlockService {
	return &blockService{s.ledger.GetBlockStore()}
}

func (s *service) TransactionService() TransactionService {
	return newTransactionService(s.nodeID, s.log, s.marshaler, s.network, s.ledger, s.txPool)
}

func (s *service) WalletService() WalletService {
	return NewWalletService(s.w)
}

func (s *service) ContractService() ContractService {
	return NewContractService(s.log, s.marshaler, s.StateQueryService(), s.TransactionService(), s.WalletService())
}
