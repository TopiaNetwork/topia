package chain

import (
	"errors"
	"fmt"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_Chain = "chain"

const (
	ChainID_Key           = "chainid"
	NetworkType_Key       = "networktype"
	LatestBlock_Key       = "latestblock"
	LatestBlockResult_Key = "latestblockresult"
)

type LedgerStateUpdater interface {
	UpdateState(state tpcmm.LedgerState)
}

type ChainState interface {
	ChainID() tpchaintypes.ChainID

	NetworkType() tpcmm.NetworkType

	GetChainRoot() ([]byte, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetLatestBlockResult() (*tpchaintypes.BlockResult, error)

	SetChainID(chainID tpchaintypes.ChainID) error

	SetNetworkType(netType tpcmm.NetworkType) error

	SetLatestBlock(block *tpchaintypes.Block) error

	SetLatestBlockResult(blockResult *tpchaintypes.BlockResult) error
}

type chainState struct {
	tplgss.StateStore
	lgUpdater LedgerStateUpdater
}

func NewChainStore(stateStore tplgss.StateStore, lgUpdater LedgerStateUpdater, cacheSize int) ChainState {
	stateStore.AddNamedStateStore(StateStore_Name_Chain, cacheSize)
	return &chainState{
		StateStore: stateStore,
		lgUpdater:  lgUpdater,
	}
}

func (cs *chainState) ChainID() tpchaintypes.ChainID {
	chainIDBytes, err := cs.GetStateData(StateStore_Name_Chain, []byte(ChainID_Key))
	if err != nil || chainIDBytes == nil {
		return tpchaintypes.ChainID_Empty
	}

	return tpchaintypes.ChainID(chainIDBytes)
}

func (cs *chainState) NetworkType() tpcmm.NetworkType {
	netTypeBytes, err := cs.GetStateData(StateStore_Name_Chain, []byte(NetworkType_Key))
	if err != nil || netTypeBytes == nil {
		return tpcmm.NetworkType_Unknown
	}

	return tpcmm.NetworkType(netTypeBytes[0])
}

func (cs *chainState) GetChainRoot() ([]byte, error) {
	return cs.Root(StateStore_Name_Chain)
}

func (cs *chainState) GetLatestBlock() (*tpchaintypes.Block, error) {
	blockBytes, err := cs.GetStateData(StateStore_Name_Chain, []byte(LatestBlock_Key))
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
	blockRSBytes, err := cs.GetStateData(StateStore_Name_Chain, []byte(LatestBlockResult_Key))
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

func (cs *chainState) SetChainID(chainID tpchaintypes.ChainID) error {
	cID := cs.ChainID()
	if cID != tpchaintypes.ChainID_Empty {
		return fmt.Errorf("Have set chain id %s", cID)
	}

	return cs.Put(StateStore_Name_Chain, []byte(ChainID_Key), []byte(chainID))
}

func (cs *chainState) SetNetworkType(netType tpcmm.NetworkType) error {
	nType := cs.NetworkType()
	if nType != tpcmm.NetworkType_Unknown {
		return fmt.Errorf("Have set net type %s", nType.String())
	}

	return cs.Put(StateStore_Name_Chain, []byte(ChainID_Key), []byte{byte(netType)})
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

	isExist, _ := cs.Exists(StateStore_Name_Chain, []byte(LatestBlock_Key))
	if isExist {
		err = cs.Update(StateStore_Name_Chain, []byte(LatestBlock_Key), blkBytes)

	} else {
		err = cs.Put(StateStore_Name_Chain, []byte(LatestBlock_Key), blkBytes)
	}

	if err == nil && block.Head.Height >= 2 {
		cs.lgUpdater.UpdateState(tpcmm.LedgerState_AutoInc)
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

	isExist, _ := cs.Exists(StateStore_Name_Chain, []byte(LatestBlock_Key))
	if isExist {
		return cs.Update(StateStore_Name_Chain, []byte(LatestBlockResult_Key), blkRSBytes)
	} else {
		return cs.Put(StateStore_Name_Chain, []byte(LatestBlockResult_Key), blkRSBytes)
	}
}
