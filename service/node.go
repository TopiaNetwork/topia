package service

import "context"

type Node struct {
}

func (n *Node) NumCheckpointsFromAccountState(ctx context.Context) uint32 {
	panic("implement me")
}

func (n *Node) NumCheckpointsFromPeerState(ctx context.Context) (uint32, error) {
	panic("implement me")
}
