package api

import (
	rpc "github.com/TopiaNetwork/topia/api/rpc"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type TPApi struct {
	log       tplog.Logger
	level     tplogcmm.LogLevel
	rpcServer *rpc.Server
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

}

func (api *TPApi) Start() {
	api.rpcServer.Start()
	api.log.Info("api started")
}
