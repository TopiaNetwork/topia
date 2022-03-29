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
	P2PProtocolPrefix         = "/universal"
	AsyncSendProtocolID       = P2PProtocolPrefix + "/asyncsend/0.0.1"
	SyncProtocolID_Block      = P2PProtocolPrefix + "/sync/blk/0.0.1"
	SyncProtocolID_Msg        = P2PProtocolPrefix + "/sync/msg/0.0.1"
	HeatBeatPtotocolID        = P2PProtocolPrefix + "/hearbeat/0.0.1"
	P2PProtocolExecutePrefix  = "/universal/execute"
	ForwardExecute_Msg        = P2PProtocolExecutePrefix + "/forward/msg/0.0.1"
	P2PProtocolProposePrefix  = "/universal/propose"
	ForwardPropose_Msg        = P2PProtocolProposePrefix + "/forward/msg/0.0.1"
	P2PProtocolValidatePrefix = "/universal/validate"
	FrowardValidate_Msg       = P2PProtocolValidatePrefix + "/forward/msg/0.0.1"
)
