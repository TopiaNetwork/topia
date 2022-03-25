package service

import (
	"context"
	types2 "github.com/TopiaNetwork/topia/chain/types"
)

type BlockChain struct {
}

func (bc *BlockChain) BlockByHeight(ctx context.Context, height uint64) (*types2.Block, error) {
	panic("implement me")
}

func (bc *BlockChain) BlockByHash(ctx context.Context, blockHash types2.BlockHash) (*types2.Block, error) {
	panic("implement me")
}
