package service

import (
	"context"
	"github.com/TopiaNetwork/topia/chain/types"
)

type BlockChain struct {
}

func (bc *BlockChain) BlockByHeight(ctx context.Context, height uint64) (*types.Block, error) {
	panic("implement me")
}

func (bc *BlockChain) BlockByHash(ctx context.Context, blockHash types.BlockHash) (*types.Block, error) {
	panic("implement me")
}

