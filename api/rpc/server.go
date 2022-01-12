package rpc

import (
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type RPCServer struct {
	log            tplog.Logger
	rpcMethodTable map[string]*RPCMethod
}

func NewRPCServer(level tplogcmm.LogLevel, log tplog.Logger) *RPCServer {
	return &RPCServer{
		log:            log,
		rpcMethodTable: make(map[string]*RPCMethod),
	}
}

func (s *RPCServer) RegisterRpcMethodTable(rpcMethodTable map[string]*RPCMethod) {
	for k, v := range rpcMethodTable {
		s.rpcMethodTable[k] = v
	}
}

func (s *RPCServer) Start() {

}
