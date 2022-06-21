package service

import (
	"math/big"
	"reflect"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

var _proxyField = "ProxyObject"

type StateQueryService interface {
	ChainID() tpchaintypes.ChainID

	NetworkType() common.NetworkType

	GetAccount(addr tpcrtypes.Address) (*tpacc.Account, error)

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

	GetAllAccounts() ([]*tpacc.Account, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetLatestBlockResult() (*tpchaintypes.BlockResult, error)

	GetAllConsensusNodeIDs() ([]string, error)

	GetNode(nodeID string) (*common.NodeInfo, error)

	GetTotalWeight() (uint64, error)

	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)

	GetInactiveNodeIDs() ([]string, error)

	GetAllActiveExecutors() ([]*common.NodeInfo, error)

	GetAllActiveProposers() ([]*common.NodeInfo, error)

	GetAllActiveValidators() ([]*common.NodeInfo, error)

	GetAllInactiveNodes() ([]*common.NodeInfo, error)

	GetNodeWeight(nodeID string) (uint64, error)

	GetActiveExecutorsTotalWeight() (uint64, error)

	GetActiveProposersTotalWeight() (uint64, error)

	GetActiveValidatorsTotalWeight() (uint64, error)

	GetLatestEpoch() (*common.EpochInfo, error)

	StateRoot() ([]byte, error)

	StateLatestVersion() (uint64, error)

	StateVersions() ([]uint64, error)

	PendingStateStore() int32
}

func GetProxyObjects(in interface{}) []interface{} {
	return getAllProxyObjectsSub(reflect.ValueOf(in).Elem())
}

func getAllProxyObjectsSub(rv reflect.Value) []interface{} {
	var out []interface{}

	internal := rv.FieldByName(_proxyField)
	ii := internal.Addr().Interface()
	out = append(out, ii)

	for i := 0; i < rv.NumField(); i++ {
		if rv.Type().Field(i).Name == _proxyField {
			continue
		}

		sub := getAllProxyObjectsSub(rv.Field(i))

		out = append(out, sub...)
	}

	return out
}

func stateQueryProxy(log tplog.Logger, ledger ledger.Ledger, proxyObject interface{}, version *uint64) {
	proxyObjs := GetProxyObjects(proxyObject)
	for _, proxy := range proxyObjs {
		proxyEle := reflect.ValueOf(proxy).Elem()
		for f := 0; f < proxyEle.NumField(); f++ {
			field := proxyEle.Type().Field(f)

			proxyEle.Field(f).Set(reflect.MakeFunc(field.Type, func(args []reflect.Value) (results []reflect.Value) {
				var compStateRN state.CompositionStateReadonly
				if version == nil {
					compStateRN = state.CreateCompositionStateReadonly(log, ledger)
				} else {
					compStateRN = state.CreateCompositionStateReadonlyAt(log, ledger, *version)
				}
				defer compStateRN.Stop()

				csRN := reflect.ValueOf(compStateRN)
				csFn := csRN.MethodByName(field.Name)
				return csFn.Call(args)
			}))
		}
	}
}

type stateQueryProxyObject struct {
	ProxyObject struct {
		ChainID func() tpchaintypes.ChainID

		NetworkType func() common.NetworkType

		GetAccount func(addr tpcrtypes.Address) (*tpacc.Account, error)

		GetNonce func(addr tpcrtypes.Address) (uint64, error)

		GetBalance func(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

		GetAllAccounts func() ([]*tpacc.Account, error)

		GetLatestBlock func() (*tpchaintypes.Block, error)

		GetLatestBlockResult func() (*tpchaintypes.BlockResult, error)

		GetAllConsensusNodeIDs func() ([]string, error)

		GetNode func(nodeID string) (*common.NodeInfo, error)

		GetTotalWeight func() (uint64, error)

		GetActiveExecutorIDs func() ([]string, error)

		GetActiveProposerIDs func() ([]string, error)

		GetActiveValidatorIDs func() ([]string, error)

		GetInactiveNodeIDs func() ([]string, error)

		GetAllActiveExecutors func() ([]*common.NodeInfo, error)

		GetAllActiveProposers func() ([]*common.NodeInfo, error)

		GetAllActiveValidators func() ([]*common.NodeInfo, error)

		GetAllInactiveNodes func() ([]*common.NodeInfo, error)

		GetNodeWeight func(nodeID string) (uint64, error)

		GetActiveExecutorsTotalWeight func() (uint64, error)

		GetActiveProposersTotalWeight func() (uint64, error)

		GetActiveValidatorsTotalWeight func() (uint64, error)

		GetLatestEpoch func() (*common.EpochInfo, error)

		StateRoot func() ([]byte, error)

		StateLatestVersion func() (uint64, error)

		StateVersions func() ([]uint64, error)

		PendingStateStore func() int32
	}
}

func (proxy *stateQueryProxyObject) ChainID() tpchaintypes.ChainID {
	return proxy.ProxyObject.ChainID()
}

func (proxy *stateQueryProxyObject) NetworkType() common.NetworkType {
	return proxy.ProxyObject.NetworkType()
}

func (proxy *stateQueryProxyObject) GetAccount(addr tpcrtypes.Address) (*tpacc.Account, error) {
	return proxy.ProxyObject.GetAccount(addr)
}

func (proxy *stateQueryProxyObject) GetNonce(addr tpcrtypes.Address) (uint64, error) {
	return proxy.ProxyObject.GetNonce(addr)
}

func (proxy *stateQueryProxyObject) GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error) {
	return proxy.ProxyObject.GetBalance(addr, symbol)
}

func (proxy *stateQueryProxyObject) GetAllAccounts() ([]*tpacc.Account, error) {
	return proxy.ProxyObject.GetAllAccounts()
}

func (proxy *stateQueryProxyObject) GetLatestBlock() (*tpchaintypes.Block, error) {
	return proxy.ProxyObject.GetLatestBlock()
}

func (proxy *stateQueryProxyObject) GetLatestBlockResult() (*tpchaintypes.BlockResult, error) {
	return proxy.ProxyObject.GetLatestBlockResult()
}

func (proxy *stateQueryProxyObject) GetAllConsensusNodeIDs() ([]string, error) {
	return proxy.ProxyObject.GetAllConsensusNodeIDs()
}

func (proxy *stateQueryProxyObject) GetNode(nodeID string) (*common.NodeInfo, error) {
	return proxy.ProxyObject.GetNode(nodeID)
}

func (proxy *stateQueryProxyObject) GetTotalWeight() (uint64, error) {
	return proxy.ProxyObject.GetTotalWeight()
}

func (proxy *stateQueryProxyObject) GetActiveExecutorIDs() ([]string, error) {
	return proxy.ProxyObject.GetActiveExecutorIDs()
}

func (proxy *stateQueryProxyObject) GetActiveProposerIDs() ([]string, error) {
	return proxy.ProxyObject.GetActiveProposerIDs()
}

func (proxy *stateQueryProxyObject) GetActiveValidatorIDs() ([]string, error) {
	return proxy.ProxyObject.GetActiveValidatorIDs()
}

func (proxy *stateQueryProxyObject) GetInactiveNodeIDs() ([]string, error) {
	return proxy.ProxyObject.GetInactiveNodeIDs()
}

func (proxy *stateQueryProxyObject) GetAllActiveExecutors() ([]*common.NodeInfo, error) {
	return proxy.ProxyObject.GetAllActiveExecutors()
}

func (proxy *stateQueryProxyObject) GetAllActiveProposers() ([]*common.NodeInfo, error) {
	return proxy.ProxyObject.GetAllActiveProposers()
}

func (proxy *stateQueryProxyObject) GetAllActiveValidators() ([]*common.NodeInfo, error) {
	return proxy.ProxyObject.GetAllActiveValidators()
}

func (proxy *stateQueryProxyObject) GetAllInactiveNodes() ([]*common.NodeInfo, error) {
	return proxy.ProxyObject.GetAllInactiveNodes()
}

func (proxy *stateQueryProxyObject) GetNodeWeight(nodeID string) (uint64, error) {
	return proxy.ProxyObject.GetNodeWeight(nodeID)
}

func (proxy *stateQueryProxyObject) GetActiveExecutorsTotalWeight() (uint64, error) {
	return proxy.ProxyObject.GetActiveExecutorsTotalWeight()
}

func (proxy *stateQueryProxyObject) GetActiveProposersTotalWeight() (uint64, error) {
	return proxy.ProxyObject.GetActiveProposersTotalWeight()
}

func (proxy *stateQueryProxyObject) GetActiveValidatorsTotalWeight() (uint64, error) {
	return proxy.ProxyObject.GetActiveValidatorsTotalWeight()
}

func (proxy *stateQueryProxyObject) GetLatestEpoch() (*common.EpochInfo, error) {
	return proxy.ProxyObject.GetLatestEpoch()
}

func (proxy *stateQueryProxyObject) StateRoot() ([]byte, error) {
	return proxy.ProxyObject.StateRoot()
}

func (proxy *stateQueryProxyObject) StateLatestVersion() (uint64, error) {
	return proxy.ProxyObject.StateLatestVersion()
}

func (proxy *stateQueryProxyObject) StateVersions() ([]uint64, error) {
	return proxy.ProxyObject.StateVersions()
}

func (proxy *stateQueryProxyObject) PendingStateStore() int32 {
	return proxy.ProxyObject.PendingStateStore()
}
