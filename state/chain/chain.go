package chain

import (
	"errors"

	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/ledger"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
	tpnet "github.com/TopiaNetwork/topia/network"
)

const StateStore_Name = "chain"

const (
	ChainID_Key           = "chainid"
	NetworkType_Key       = "networktype"
	LatestBlock_Key       = "latestblock"
	LatestBlockResult_Key = "latestblockresult"
)

type ChainState interface {
	ChainID() chain.ChainID

	NetworkType() tpnet.NetworkType

	GetChainRoot() ([]byte, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetLatestBlockResult() (*tpchaintypes.BlockResult, error)

	SetLatestBlock(block *tpchaintypes.Block) error

	SetLatestBlockResult(blockResult *tpchaintypes.BlockResult) error
}

type chainState struct {
	tplgss.StateStore
	ledger ledger.Ledger
}

func NewChainStore(stateStore tplgss.StateStore, ledger ledger.Ledger) ChainState {
	stateStore.AddNamedStateStore(StateStore_Name)
	return &chainState{
		StateStore: stateStore,
	}
}

func (cs *chainState) ChainID() chain.ChainID {
	chainIDBytes, _, err := cs.GetState(StateStore_Name, []byte(ChainID_Key))
	if err != nil {
		return chain.ChainID_Empty
	}

	return chain.ChainID(chainIDBytes)
}

func (cs *chainState) NetworkType() tpnet.NetworkType {
	netTypeBytes, _, err := cs.GetState(StateStore_Name, []byte(NetworkType_Key))
	if err != nil {
		return tpnet.NetworkType_Unknown
	}

	return tpnet.NetworkType(netTypeBytes[0])
}

func (cs *chainState) GetChainRoot() ([]byte, error) {
	return cs.Root(StateStore_Name)
}

func (cs *chainState) GetLatestBlock() (*tpchaintypes.Block, error) {
	blockBytes, _, err := cs.GetState(StateStore_Name, []byte(LatestBlock_Key))
	if err != nil {
		return nil, err
	}

	var block tpchaintypes.Block
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	err = marshaler.Unmarshal(blockBytes, &block)
	if err != nil {
		return nil, err
	}

	return &block, nil
}

func (cs *chainState) GetLatestBlockResult() (*tpchaintypes.BlockResult, error) {
	blockRSBytes, _, err := cs.GetState(StateStore_Name, []byte(LatestBlockResult_Key))
	if err != nil {
		return nil, err
	}

	var blockRS tpchaintypes.BlockResult
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	err = marshaler.Unmarshal(blockRSBytes, &blockRS)
	if err != nil {
		return nil, err
	}

	return &blockRS, nil
}

func (cs *chainState) SetLatestBlock(block *tpchaintypes.Block) error {
	if block == nil {
		return errors.New("Nil block")
	}

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	blkBytes, err := marshaler.Marshal(block)
	if err != nil {
		return err
	}

	isExist, _ := cs.Exists(StateStore_Name, []byte(LatestBlock_Key))
	if isExist {
		err = cs.Update(StateStore_Name, []byte(LatestBlock_Key), blkBytes)

	} else {
		err = cs.Put(StateStore_Name, []byte(LatestBlock_Key), blkBytes)
	}

	if err == nil && block.Head.Height >= 2 {
		cs.ledger.UpdateState(ledger.LedgerState_AutoInc)
	}

	return err
}

func (cs *chainState) SetLatestBlockResult(blockResult *tpchaintypes.BlockResult) error {
	if blockResult == nil {
		return errors.New("Nil block result")
	}

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	blkRSBytes, err := marshaler.Marshal(blockResult)
	if err != nil {
		return err
	}

	isExist, _ := cs.Exists(StateStore_Name, []byte(LatestBlock_Key))
	if isExist {
		return cs.Update(StateStore_Name, []byte(LatestBlockResult_Key), blkRSBytes)
	} else {
		return cs.Put(StateStore_Name, []byte(LatestBlockResult_Key), blkRSBytes)
	}
}
