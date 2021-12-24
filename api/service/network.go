package service

import (
	"context"
)

type NetworkParameters struct {
	ChainID string
}

type Network struct {
}

func (net *Network) Ping(ctx context.Context) error {
	panic("implement me")
}

func (net *Network) NetworkParam(ctx context.Context) (*NetworkParameters, error) {
	panic("implement me")
}
