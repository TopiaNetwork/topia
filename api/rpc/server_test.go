package rpc

import (
	"github.com/coocood/freecache"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRegisterMethod(t *testing.T) {
	server := newTestHttpServer()

	var funcExample1 = func(i int, s string, f float64) (int, string, float64) {
		return i, s, f
	}

	err := server.Register(funcExample1, "funcExample1", Read, false, 10, 10*time.Second)
	assert.Nil(t, err)
}

var (
	testReadToken    = "awef"
	testWriteToken   = "af2e2"
	testSignToken    = "vesfa"
	testManagerToken = "grsfa"
)

type testI interface {
	TestMethod1()
	TestMethod2()
}
type testStruct struct {
	ProxyObj struct {
		TestMethod1 func() `grant:"write"`
		TestMethod2 func() `grant:"manager"`
	}
}

func (ts *testStruct) TestMethod1() {}
func (ts *testStruct) TestMethod2() {}

func TestRegisterStruct(t *testing.T) {
	server := newTestHttpServer()

	var interfaceToRegister = &testStruct{}
	err := server.Register(interfaceToRegister, "someName", Read, false, -1, 10*time.Second)
	assert.Nil(t, err)
	assert.Equal(t, Write, server.methodMap["someName_TestMethod1"].authLevel)
	assert.Equal(t, Manager, server.methodMap["someName_TestMethod2"].authLevel)

}

func TestVerifyAuth(t *testing.T) {
	server := newTestHttpServer()

	var funcRead = func() {}
	var funcWrite = func() {}
	var funcSign = func() {}
	var funcManager = func() {}

	err := server.registerMethod(funcRead, "funcRead", Read, false, -1, 10*time.Second)
	assert.Nil(t, err)
	err = server.registerMethod(funcWrite, "funcWrite", Write, false, -1, 10*time.Second)
	assert.Nil(t, err)
	err = server.registerMethod(funcSign, "funcSign", Sign, false, -1, 10*time.Second)
	assert.Nil(t, err)
	err = server.registerMethod(funcManager, "funcManager", Manager, false, -1, 10*time.Second)
	assert.Nil(t, err)

	pass, err := server.Verify(testWriteToken, "funcWrite")
	assert.Nil(t, err)
	assert.Equal(t, true, pass)

	pass, err = server.Verify(testWriteToken, "funcRead")
	assert.Nil(t, err)
	assert.Equal(t, true, pass)

	pass, err = server.Verify(testWriteToken, "funcManager")
	assert.Nil(t, err)
	assert.Equal(t, false, pass)

	pass, err = server.Verify("notExistToken", "funcManager")
	assert.Nil(t, err)
	assert.Equal(t, false, pass)

	pass, err = server.Verify(testManagerToken, "notExistFunc")
	assert.NotNil(t, err)
	assert.Equal(t, false, pass)
}

func TestShutDownServer(t *testing.T) {
	server := newTestHttpServer()

	go server.Start()
	time.Sleep(2 * time.Second)
	err := server.shutDown()
	assert.Nil(t, err)
}

var testServerHost = "localhost:8199"

func newTestHttpServer() *Server {
	m := make(map[AuthLevel]string, 5)
	m[None] = ""
	m[Read] = testReadToken
	m[Write] = testWriteToken
	m[Sign] = testSignToken
	m[Manager] = testManagerToken

	var auth = NewAuthObject(m)
	c := freecache.NewCache(2 * 1024 * 1024)

	return NewServer(
		testServerHost,
		SetAUTH(auth),
		SetCache(c),
	)
}

func newTestWebsocketServer() *Server {
	m := make(map[AuthLevel]string, 5)
	m[None] = ""
	m[Read] = testReadToken
	m[Write] = testWriteToken
	m[Sign] = testSignToken
	m[Manager] = testManagerToken

	var auth = NewAuthObject(m)
	c := freecache.NewCache(2 * 1024 * 1024)

	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	return NewServer(
		testServerHost,
		SetAUTH(auth),
		SetCache(c),
		SetWebsocket(upgrader),
	)
}
