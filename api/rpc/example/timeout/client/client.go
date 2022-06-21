package main

import (
	rpc "github.com/TopiaNetwork/topia/api/rpc"
	"github.com/gregjones/httpcache/diskcache"

	tlog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
)

func main() {
	mylog, err := tlog.CreateMainLogger(logcomm.DebugLevel, tlog.JSONFormat, tlog.StdErrOutput, "")
	if err != nil {
		panic(err)
	}

	addr := "localhost:8199"

	cache := diskcache.New("data")

	cli, _ := rpc.NewClient(
		addr, rpc.SetClientAUTH("vesfa"),
		rpc.SetClientCache(cache),
		rpc.SetClientWS(512, "10s"),
	)

	inArgs := make([]interface{}, 3)
	inArgs[0] = 1
	inArgs[1] = "test"
	inArgs[2] = 10.0

	f1 := func(inArgs []interface{}) {
		methodName := "MyTest"
		res, err := cli.Call(methodName, inArgs)
		if err != nil {
			mylog.Info("1" + err.Error())
		} else {
			mylog.Infof("%v", string(res.Payload))
		}
	}

	f2 := func(inArgs []interface{}) {
		methodName := "MyTestWithSleep"
		_, res2, err := cli.CallWithWS(methodName, inArgs)
		if err != nil {
			mylog.Info(err.Error())
		} else {
			mylog.Info(string(<-res2))
		}
	}

	f1(inArgs)
	f2(inArgs)
	// go f2(inArgs)
	// go f2(inArgs)
	// go f2(inArgs)
	// go f2(inArgs)

	// time.Sleep(5 * time.Second)
	// time.Sleep(1 * time.Second)
	// go f2(inArgs)
	// go f2(inArgs)
	// go f2(inArgs)
	// go f2(inArgs)
	// go f2(inArgs)
	for {

	}
}
