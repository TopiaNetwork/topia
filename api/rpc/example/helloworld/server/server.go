package main

import (
	"log"
	"time"

	rpc "github.com/TopiaNetwork/topia/api/rpc"
	"github.com/TopiaNetwork/topia/api/rpc/example/helloworld/public"
	tlog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	"github.com/coocood/freecache"
	"github.com/gorilla/websocket"
)

func main() {
	mylog, err := tlog.CreateMainLogger(logcomm.DebugLevel, tlog.JSONFormat, tlog.StdErrOutput, "")
	if err != nil {
		panic(err)
	}
	addr := "localhost:8199"
	m := make(map[byte]string, 5)
	m[rpc.None] = ""
	m[rpc.Read] = "awef"
	m[rpc.Write] = "af2e2"
	m[rpc.Sign] = "vesfa"
	m[rpc.Manager] = "grsfa"

	var auth rpc.AuthObject = *rpc.NewAuthObject(m)
	c := freecache.NewCache(2 * 1024 * 1024)

	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	srv := rpc.NewServer(addr, rpc.SetAUTH(auth), rpc.SetCache(c), rpc.SetWebsocket(upgrader))
	var f func(i int, s string, f float64) (res string, e error) = public.MyTest
	e := srv.Register(f, "MyTest", 0x04, 60, 10*time.Second)
	if e != nil {
		log.Println(e)
		return
	}

	mylog.Info("service is running")
	srv.Start()
}
