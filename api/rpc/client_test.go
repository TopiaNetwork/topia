package rpc

import (
	"fmt"
	tlog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCallWithHttp(t *testing.T) {
	server := newTestHttpServer()
	var funcExample1 = func(i int, s string, f float64) (int, string, float64) {
		return i, s, f
	}
	err := server.Register(funcExample1, "funcExample1", Read, false, 10, 10*time.Second)
	assert.Nil(t, err)

	go server.Start()

	time.Sleep(3 * time.Second)

	client, err := newTestHttpClient()
	assert.Nil(t, err)

	message, err := client.callWithHttp("funcExample1", 1, "2", 3.0)
	assert.Nil(t, err)
	assert.Equal(t, NoErr, message.ErrMsg.Errtype)
}

func TestCallWithWebsocket(t *testing.T) {
	server := newTestWebsocketServer()
	var funcExample1 = func(i int, s string, f float64) (int, string, float64) {
		return i, s, f
	}
	err := server.Register(funcExample1, "funcExample1", Read, false, 10, 10*time.Second)
	assert.Nil(t, err)

	go server.Start()

	time.Sleep(3 * time.Second)

	client, err := newTestWebsocketClient()
	assert.Nil(t, err)

	message, err := client.callWithWS("funcExample1", 1, "2", 3.0)
	assert.Nil(t, err)
	assert.Equal(t, NoErr, message.ErrMsg.Errtype)
}

func TestCloseServer(t *testing.T) {
	server := newTestHttpServer()

	go server.Start()

	time.Sleep(3 * time.Second)

	client, err := newTestHttpClient()
	assert.Nil(t, err)

	err = client.CloseServer()
	assert.Nil(t, err)
}

func newTestHttpClient() (client *Client, err error) {
	serverURL := fmt.Sprintf("http://%s", testServerHost)
	cache := diskcache.New("test_cache")

	return NewClient(
		serverURL,
		SetClientCache(cache),
		SetClientAUTH(testManagerToken), // highest auth
	)
}

func newTestWebsocketClient() (client *Client, err error) {
	mylog, err := tlog.CreateMainLogger(logcomm.DebugLevel, tlog.JSONFormat, tlog.StdErrOutput, "")
	if err != nil {
		panic(err)
	}
	serverURL := fmt.Sprintf("ws://%s", testServerHost)
	cache := diskcache.New("test_cache")

	return NewClient(
		serverURL,
		SetClientCache(cache),
		SetClientAUTH(testManagerToken), // highest auth
		SetClientWS(512, "10s", mylog),
	)
}
