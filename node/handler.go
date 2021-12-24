package node

import (
	tplog "github.com/TopiaNetwork/topia/log"
)

type NodeHandler interface {
	ProcessDKG(msg *DKGMessage) error
}

type nodeHandler struct {
	log tplog.Logger
}

func NewNodeHandler(log tplog.Logger) *nodeHandler {
	return &nodeHandler{
		log: log,
	}
}

func (handler *nodeHandler) ProcessDKG(msg *DKGMessage) error {
	panic("implement me")
}
