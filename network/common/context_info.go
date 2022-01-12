package common

type NetContextKey int

const (
	NetContextKey_Unknown NetContextKey = iota
	NetContextKey_PeerList
	NetContextKey_RouteStrategy
	NetContextKey_RespThreshold
	NetContextKey_BOOTNODES
)

type RouteStrategy int

const (
	RouteStrategy_Unknown RouteStrategy = iota
	RouteStrategy_Default
	RouteStrategy_NearestBucket
	RouteStrategy_BucketsWithFactor
)
