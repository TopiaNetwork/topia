package api

import (
	"github.com/TopiaNetwork/topia/api/rpc"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	service2 "github.com/TopiaNetwork/topia/service"
)

type TPApi struct {
	log       tplog.Logger
	level     tplogcmm.LogLevel
	rpcServer *rpc.RPCServer
}

func NewTPApi(level tplogcmm.LogLevel, log tplog.Logger) *TPApi {
	apiLog := tplog.CreateModuleLogger(level, "api", log)

	tpAPI := &TPApi{
		log: apiLog,
	}
	tpAPI.init()

	return tpAPI
}

func (api *TPApi) init() {

	bcService := &service2.BlockChain{}
	netService := &service2.Network{}
	nodeService := &service2.Node{}
	txService := &service2.Transaction{}

	api.rpcServer = rpc.NewRPCServer(api.level, api.log)

	api.rpcServer.RegisterRpcMethodTable(map[string]*rpc.RPCMethod{
		"block_by_height": rpc.NewRPCFunc(bcService.BlockByHeight, "height", rpc.MethodPerm_Read, true),
		"block_by_hash":   rpc.NewRPCFunc(bcService.BlockByHash, "blockHash", rpc.MethodPerm_Read, true),

		"ping":          rpc.NewRPCFunc(netService.Ping, "", rpc.MethodPerm_Read|rpc.MethodPerm_Write, false),
		"network_param": rpc.NewRPCFunc(netService.NetworkParam, "", rpc.MethodPerm_Read, true),

		"numcheckpoints_from_accountstate": rpc.NewRPCFunc(nodeService.NumCheckpointsFromAccountState, "", rpc.MethodPerm_Read, true),
		"numcheckpoints_from_peerstate":    rpc.NewRPCFunc(nodeService.NumCheckpointsFromPeerState, "", rpc.MethodPerm_Read, true),

		"send_transaction":  rpc.NewRPCFunc(txService.SendTransaction, "tran", rpc.MethodPerm_Read|rpc.MethodPerm_Write, false),
		"transaction_by_id": rpc.NewRPCFunc(txService.TransactionByID, "txHash", rpc.MethodPerm_Read, true),
	})
}

func (api *TPApi) Start() {
	api.rpcServer.Start()
	api.log.Info("api started")
}
