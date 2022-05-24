package main

import (
	"log"

	rpc "github.com/TopiaNetwork/topia/api/rpc"
	"github.com/TopiaNetwork/topia/api/rpc/example/helloworld/public"
	"github.com/coocood/freecache"
)

func main() {
	addr := "localhost:8199"
	m := make([]string, 5)
	m[rpc.None] = ""
	m[rpc.Read] = "awef"
	m[rpc.Write] = "af2e2"
	m[rpc.Sign] = "vesfa"
	m[rpc.Manager] = "grsfa"

	var auth rpc.AuthObject = *rpc.NewAuthObject(m)
	c := freecache.NewCache(2 * 1024 * 1024)
	srv := rpc.NewServer(addr, rpc.SetAUTH(auth), rpc.SetCache(c))
	var f func(i int, s string, f float64) (res string, e error) = public.MyTest
	e := srv.Register(f, "MyTest", 2, 60)
	if e != nil {
		log.Println(e)
		return
	}

	log.Println("service is running")
	srv.Run()
}
