package protocol

import "time"

const (
	_  = iota
	kb = 1 << (10 * iota)
	mb
	gb
)

const (
	PubSubMaxMsgSize = 5 * mb
	StreamMaxMsgSize = gb
	WriteReqDeadline = 5 * time.Second
	ReadResDeadline  = WriteReqDeadline
	ReadResMinSpeed  = 50 << 10
	WaitRespTimeout  = 30 * time.Second
)

const (
	P2PProtocolPrefix          = "/topia"
	AsyncSendProtocolID        = P2PProtocolPrefix + "/asyncsend/0.0.1"
	SyncProtocolID_Block       = P2PProtocolPrefix + "/sync/blk/0.0.1"
	SyncProtocolID_State       = P2PProtocolPrefix + "/sync/state/0.0.1"
	SyncProtocolId_Epoch       = P2PProtocolPrefix + "/sync/epoch/0.0.1"
	SyncProtocolId_Node        = P2PProtocolPrefix + "/sync/node/0.0.1"
	SyncProtocolId_Account     = P2PProtocolPrefix + "/sync/account/0.0.1"
	SyncProtocolId_Chain       = P2PProtocolPrefix + "/sync/chain/0.0.1"
	SyncProtocolID_Msg         = P2PProtocolPrefix + "/sync/msg/0.0.1"
	PubSubProtocolID_BlockInfo = P2PProtocolPrefix + "/pubsub/blkinfo/0.0.1"
	PubSubProtocolID_Msgs      = P2PProtocolPrefix + "/pubsub/msgs/0.0.1"
	HeartBeatPtotocolID        = P2PProtocolPrefix + "/hearbeat/0.0.1"

	P2PProtocolExecutePrefix  = "/topia/execute"
	ForwardExecute_Msg        = P2PProtocolExecutePrefix + "/forward/msg/0.0.1"
	ForwardExecute_SyncMsg    = P2PProtocolExecutePrefix + "/forward/sync/msg/0.0.1"
	ForwardExecute_SyncTx     = P2PProtocolExecutePrefix + "/forward/sync/tx/0.0.1"
	P2PProtocolProposePrefix  = "/topia/propose"
	ForwardPropose_Msg        = P2PProtocolProposePrefix + "/forward/msg/0.0.1"
	P2PProtocolValidatePrefix = "/topia/validate"
	FrowardValidate_Msg       = P2PProtocolValidatePrefix + "/forward/msg/0.0.1"
)
