package common

type NetContextKey int

const (
	NetContextKey_Unknown NetContextKey = iota
	NetContextKey_PeerList
	NetContextKey_RouteStrategy
	NetContextKey_RespThreshold
	NetContextKey_BOOTNODES
	NetContextKey_BOOTNODES_EXECUTE
	NetContextKey_BOOTNODES_PROPOSE
	NetContextKey_BOOTNODES_VALIDATE
)

type RouteStrategy int

const (
	RouteStrategy_Unknown RouteStrategy = iota
	RouteStrategy_Default
	RouteStrategy_NearestBucket
	RouteStrategy_BucketsWithFactor
)
