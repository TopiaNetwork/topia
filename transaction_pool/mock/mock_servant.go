// Code generated by MockGen. DO NOT EDIT.
// Source: servant.go

// Package mock_transactionpool is a generated GoMock package.
package mock_transactionpool

import (
	big "math/big"
	reflect "reflect"

	account "github.com/TopiaNetwork/topia/account"
	types "github.com/TopiaNetwork/topia/common/types"
	p2p "github.com/TopiaNetwork/topia/network/p2p"
	transaction "github.com/TopiaNetwork/topia/transaction"
	transaction_pool "github.com/TopiaNetwork/topia/transaction_pool"
	gomock "github.com/golang/mock/gomock"
)

// MockTransactionPoolServant is a mock of TransactionPoolServant interface.
type MockTransactionPoolServant struct {
	ctrl     *gomock.Controller
	recorder *MockTransactionPoolServantMockRecorder
}

// MockTransactionPoolServantMockRecorder is the mock recorder for MockTransactionPoolServant.
type MockTransactionPoolServantMockRecorder struct {
	mock *MockTransactionPoolServant
}

// NewMockTransactionPoolServant creates a new mock instance.
func NewMockTransactionPoolServant(ctrl *gomock.Controller) *MockTransactionPoolServant {
	mock := &MockTransactionPoolServant{ctrl: ctrl}
	mock.recorder = &MockTransactionPoolServantMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransactionPoolServant) EXPECT() *MockTransactionPoolServantMockRecorder {
	return m.recorder
}

// CurrentBlock mocks base method.
func (m *MockTransactionPoolServant) CurrentBlock() *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CurrentBlock")
	ret0, _ := ret[0].(*types.Block)
	return ret0
}

// CurrentBlock indicates an expected call of CurrentBlock.
func (mr *MockTransactionPoolServantMockRecorder) CurrentBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CurrentBlock", reflect.TypeOf((*MockTransactionPoolServant)(nil).CurrentBlock))
}

// EstimateTxCost mocks base method.
func (m *MockTransactionPoolServant) EstimateTxCost(tx *transaction.Transaction) *big.Int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EstimateTxCost", tx)
	ret0, _ := ret[0].(*big.Int)
	return ret0
}

// EstimateTxCost indicates an expected call of EstimateTxCost.
func (mr *MockTransactionPoolServantMockRecorder) EstimateTxCost(tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EstimateTxCost", reflect.TypeOf((*MockTransactionPoolServant)(nil).EstimateTxCost), tx)
}

// EstimateTxGas mocks base method.
func (m *MockTransactionPoolServant) EstimateTxGas(tx *transaction.Transaction) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EstimateTxGas", tx)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// EstimateTxGas indicates an expected call of EstimateTxGas.
func (mr *MockTransactionPoolServantMockRecorder) EstimateTxGas(tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EstimateTxGas", reflect.TypeOf((*MockTransactionPoolServant)(nil).EstimateTxGas), tx)
}

// GetBlock mocks base method.
func (m *MockTransactionPoolServant) GetBlock(hash types.BlockHash, num uint64) *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlock", hash, num)
	ret0, _ := ret[0].(*types.Block)
	return ret0
}

// GetBlock indicates an expected call of GetBlock.
func (mr *MockTransactionPoolServantMockRecorder) GetBlock(hash, num interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlock", reflect.TypeOf((*MockTransactionPoolServant)(nil).GetBlock), hash, num)
}

// StateAt mocks base method.
func (m *MockTransactionPoolServant) StateAt(root types.BlockHash) (*transaction_pool.StatePoolDB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StateAt", root)
	ret0, _ := ret[0].(*transaction_pool.StatePoolDB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StateAt indicates an expected call of StateAt.
func (mr *MockTransactionPoolServantMockRecorder) StateAt(root interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StateAt", reflect.TypeOf((*MockTransactionPoolServant)(nil).StateAt), root)
}

// SubChainHeadEvent mocks base method.
func (m *MockTransactionPoolServant) SubChainHeadEvent(ch chan<- transaction.ChainHeadEvent) p2p.P2PPubSubService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubChainHeadEvent", ch)
	ret0, _ := ret[0].(p2p.P2PPubSubService)
	return ret0
}

// SubChainHeadEvent indicates an expected call of SubChainHeadEvent.
func (mr *MockTransactionPoolServantMockRecorder) SubChainHeadEvent(ch interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubChainHeadEvent", reflect.TypeOf((*MockTransactionPoolServant)(nil).SubChainHeadEvent), ch)
}

// MockStatePoolDB is a mock of StatePoolDB interface.
type MockStatePoolDB struct {
	ctrl     *gomock.Controller
	recorder *MockStatePoolDBMockRecorder
}

// MockStatePoolDBMockRecorder is the mock recorder for MockStatePoolDB.
type MockStatePoolDBMockRecorder struct {
	mock *MockStatePoolDB
}

// NewMockStatePoolDB creates a new mock instance.
func NewMockStatePoolDB(ctrl *gomock.Controller) *MockStatePoolDB {
	mock := &MockStatePoolDB{ctrl: ctrl}
	mock.recorder = &MockStatePoolDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStatePoolDB) EXPECT() *MockStatePoolDBMockRecorder {
	return m.recorder
}

// GetAccount mocks base method.
func (m *MockStatePoolDB) GetAccount(addr account.Address) (*account.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccount", addr)
	ret0, _ := ret[0].(*account.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccount indicates an expected call of GetAccount.
func (mr *MockStatePoolDBMockRecorder) GetAccount(addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccount", reflect.TypeOf((*MockStatePoolDB)(nil).GetAccount), addr)
}

// GetBalance mocks base method.
func (m *MockStatePoolDB) GetBalance(addr account.Address) *big.Int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalance", addr)
	ret0, _ := ret[0].(*big.Int)
	return ret0
}

// GetBalance indicates an expected call of GetBalance.
func (mr *MockStatePoolDBMockRecorder) GetBalance(addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalance", reflect.TypeOf((*MockStatePoolDB)(nil).GetBalance), addr)
}

// GetNonce mocks base method.
func (m *MockStatePoolDB) GetNonce(addr account.Address) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNonce", addr)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetNonce indicates an expected call of GetNonce.
func (mr *MockStatePoolDBMockRecorder) GetNonce(addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNonce", reflect.TypeOf((*MockStatePoolDB)(nil).GetNonce), addr)
}
