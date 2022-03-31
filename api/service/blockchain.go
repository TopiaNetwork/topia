package service

import (
	"context"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
)

type BlockChain struct {
}

func (bc *BlockChain) BlockByHeight(ctx context.Context, height uint64) (*tpchaintypes.Block, error) {
	panic("implement me")
}

func (bc *BlockChain) BlockByHash(ctx context.Context, blockHash tpchaintypes.BlockHash) (*tpchaintypes.Block, error) {
	panic("implement me")
}

