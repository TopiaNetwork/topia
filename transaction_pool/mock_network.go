// Code generated by MockGen. DO NOT EDIT.

// Source: /Users/junwang/topia/topia/network/network.go

// Package mock_network is a generated GoMock package.
package transactionpool

import (
	context "context"
	reflect "reflect"

	actor "github.com/AsynkronIT/protoactor-go/actor"
	codec "github.com/TopiaNetwork/topia/codec"
	message "github.com/TopiaNetwork/topia/network/message"
	gomock "github.com/golang/mock/gomock"
)

// MockNetwork is a mock of Network interface.
type MockNetwork struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkMockRecorder
}

// MockNetworkMockRecorder is the mock recorder for MockNetwork.
type MockNetworkMockRecorder struct {
	mock *MockNetwork
}

// NewMockNetwork creates a new mock instance.
func NewMockNetwork(ctrl *gomock.Controller) *MockNetwork {
	mock := &MockNetwork{ctrl: ctrl}
	mock.recorder = &MockNetworkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNetwork) EXPECT() *MockNetworkMockRecorder {
	return m.recorder
}

// Connect mocks base method.
func (m *MockNetwork) Connect(listenAddr []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connect", listenAddr)
	ret0, _ := ret[0].(error)
	return ret0
}

// Connect indicates an expected call of Connect.
func (mr *MockNetworkMockRecorder) Connect(listenAddr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connect", reflect.TypeOf((*MockNetwork)(nil).Connect), listenAddr)
}

// ID mocks base method.
func (m *MockNetwork) ID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockNetworkMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockNetwork)(nil).ID))
}

// ListenAddr mocks base method.
func (m *MockNetwork) ListenAddr() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListenAddr")
	ret0, _ := ret[0].([]string)
	return ret0
}

// ListenAddr indicates an expected call of ListenAddr.
func (mr *MockNetworkMockRecorder) ListenAddr() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListenAddr", reflect.TypeOf((*MockNetwork)(nil).ListenAddr))
}

// Publish mocks base method.
func (m *MockNetwork) Publish(ctx context.Context, toModuleNames []string, topic string, data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Publish", ctx, toModuleNames, topic, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// Publish indicates an expected call of Publish.
func (mr *MockNetworkMockRecorder) Publish(ctx, toModuleNames, topic, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Publish", reflect.TypeOf((*MockNetwork)(nil).Publish), ctx, toModuleNames, topic, data)
}

// RegisterModule mocks base method.
func (m *MockNetwork) RegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterModule", moduleName, pid, marshaler)
}

// RegisterModule indicates an expected call of RegisterModule.
func (mr *MockNetworkMockRecorder) RegisterModule(moduleName, pid, marshaler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterModule", reflect.TypeOf((*MockNetwork)(nil).RegisterModule), moduleName, pid, marshaler)
}

// Send mocks base method.
func (m *MockNetwork) Send(ctx context.Context, protocolID, moduleName string, data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", ctx, protocolID, moduleName, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockNetworkMockRecorder) Send(ctx, protocolID, moduleName, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockNetwork)(nil).Send), ctx, protocolID, moduleName, data)
}

// SendWithResponse mocks base method.
func (m *MockNetwork) SendWithResponse(ctx context.Context, protocolID, moduleName string, data []byte) ([][]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendWithResponse", ctx, protocolID, moduleName, data)
	ret0, _ := ret[0].([][]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendWithResponse indicates an expected call of SendWithResponse.
func (mr *MockNetworkMockRecorder) SendWithResponse(ctx, protocolID, moduleName, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendWithResponse", reflect.TypeOf((*MockNetwork)(nil).SendWithResponse), ctx, protocolID, moduleName, data)
}

// Start mocks base method.
func (m *MockNetwork) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start.
func (mr *MockNetworkMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockNetwork)(nil).Start))
}

// Stop mocks base method.
func (m *MockNetwork) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop.
func (mr *MockNetworkMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockNetwork)(nil).Stop))
}

// Subscribe mocks base method.
func (m *MockNetwork) Subscribe(ctx context.Context, topic string, validators ...message.PubSubMessageValidator) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, topic}
	for _, a := range validators {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Subscribe", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Subscribe indicates an expected call of Subscribe.
func (mr *MockNetworkMockRecorder) Subscribe(ctx, topic interface{}, validators ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, topic}, validators...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockNetwork)(nil).Subscribe), varargs...)
}

// UnRegisterModule mocks base method.
func (m *MockNetwork) UnRegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UnRegisterModule", moduleName, pid, marshaler)
}

// UnRegisterModule indicates an expected call of UnRegisterModule.
func (mr *MockNetworkMockRecorder) UnRegisterModule(moduleName, pid, marshaler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnRegisterModule", reflect.TypeOf((*MockNetwork)(nil).UnRegisterModule), moduleName, pid, marshaler)
}

// UnSubscribe mocks base method.
func (m *MockNetwork) UnSubscribe(topic string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnSubscribe", topic)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnSubscribe indicates an expected call of UnSubscribe.
func (mr *MockNetworkMockRecorder) UnSubscribe(topic interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnSubscribe", reflect.TypeOf((*MockNetwork)(nil).UnSubscribe), topic)
}
