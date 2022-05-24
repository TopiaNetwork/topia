package main

import (
	"log"

	rpc "github.com/TopiaNetwork/topia/api/rpc"
)

func main() {

	addr := "http://localhost:8199"

	cli := rpc.NewClient(addr, rpc.SetClientAUTH("vesfa"))
	inArgs := make([]interface{}, 3)
	inArgs[0] = 1
	inArgs[1] = "test"
	inArgs[2] = 10.0
	methodName := "MyTest"
	res, err := cli.Call(methodName, inArgs)

	if err != nil {
		log.Print(err.Error())
	} else {
		log.Print(res)
	}
}
