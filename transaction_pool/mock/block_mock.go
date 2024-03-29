// Code generated by MockGen. DO NOT EDIT.
// Source: ./service/block.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	types "github.com/TopiaNetwork/topia/chain/types"
	basic "github.com/TopiaNetwork/topia/transaction/basic"
	gomock "github.com/golang/mock/gomock"
)

// MockBlockService is a mock of BlockService interface.
type MockBlockService struct {
	ctrl     *gomock.Controller
	recorder *MockBlockServiceMockRecorder
}

// MockBlockServiceMockRecorder is the mock recorder for MockBlockService.
type MockBlockServiceMockRecorder struct {
	mock *MockBlockService
}

// NewMockBlockService creates a new mock instance.
func NewMockBlockService(ctrl *gomock.Controller) *MockBlockService {
	mock := &MockBlockService{ctrl: ctrl}
	mock.recorder = &MockBlockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBlockService) EXPECT() *MockBlockServiceMockRecorder {
	return m.recorder
}

// GetBatchBlocks mocks base method.
func (m *MockBlockService) GetBatchBlocks(startBlockNum types.BlockNum, count uint64) ([]*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBatchBlocks", startBlockNum, count)
	ret0, _ := ret[0].([]*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBatchBlocks indicates an expected call of GetBatchBlocks.
func (mr *MockBlockServiceMockRecorder) GetBatchBlocks(startBlockNum, count interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBatchBlocks", reflect.TypeOf((*MockBlockService)(nil).GetBatchBlocks), startBlockNum, count)
}

// GetBlockByHash mocks base method.
func (m *MockBlockService) GetBlockByHash(blockHash types.BlockHash) (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockByHash", blockHash)
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockByHash indicates an expected call of GetBlockByHash.
func (mr *MockBlockServiceMockRecorder) GetBlockByHash(blockHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByHash", reflect.TypeOf((*MockBlockService)(nil).GetBlockByHash), blockHash)
}

// GetBlockByNumber mocks base method.
func (m *MockBlockService) GetBlockByNumber(blockNum types.BlockNum) (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockByNumber", blockNum)
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockByNumber indicates an expected call of GetBlockByNumber.
func (mr *MockBlockServiceMockRecorder) GetBlockByNumber(blockNum interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByNumber", reflect.TypeOf((*MockBlockService)(nil).GetBlockByNumber), blockNum)
}

// GetBlockByTxID mocks base method.
func (m *MockBlockService) GetBlockByTxID(txID basic.TxID) (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockByTxID", txID)
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockByTxID indicates an expected call of GetBlockByTxID.
func (mr *MockBlockServiceMockRecorder) GetBlockByTxID(txID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByTxID", reflect.TypeOf((*MockBlockService)(nil).GetBlockByTxID), txID)
}

// GetTransactionByID mocks base method.
func (m *MockBlockService) GetTransactionByID(txID basic.TxID) (*basic.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactionByID", txID)
	ret0, _ := ret[0].(*basic.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTransactionByID indicates an expected call of GetTransactionByID.
func (mr *MockBlockServiceMockRecorder) GetTransactionByID(txID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransactionByID", reflect.TypeOf((*MockBlockService)(nil).GetTransactionByID), txID)
}

// GetTransactionResultByID mocks base method.
func (m *MockBlockService) GetTransactionResultByID(txID basic.TxID) (*basic.TransactionResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactionResultByID", txID)
	ret0, _ := ret[0].(*basic.TransactionResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTransactionResultByID indicates an expected call of GetTransactionResultByID.
func (mr *MockBlockServiceMockRecorder) GetTransactionResultByID(txID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransactionResultByID", reflect.TypeOf((*MockBlockService)(nil).GetTransactionResultByID), txID)
}
