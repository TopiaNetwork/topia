package types

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"math/big"
	"strconv"
)

type GetBlockByHashRequestType struct {
	BlockHash string
}

type GetBlockResponseType struct {
	Number           uint64   `json:"number"`
	Hash             []byte   `json:"hash"`
	ParentHash       []byte   `json:"parentHash"`
	Nonce            *big.Int `json:"nonce"`
	MixHash          []byte   `json:"mixHash"`
	Sha3Uncles       []byte   `json:"sha3Uncles"`
	LogsBloom        []byte   `json:"logsBloom"`
	StateRoot        []byte   `json:"stateRoot"`
	Miner            []byte   `json:"miner"`
	Difficulty       *big.Int `json:"difficulty"`
	ExtraData        []byte   `json:"extraData"`
	Size             int      `json:"size"`
	GasLimit         []byte   `json:"gasLimit"`
	GasUsed          []byte   `json:"gasUsed"`
	Timestamp        uint64   `json:"timestamp"`
	TransactionsRoot []byte   `json:"transactionsRoot"`
	ReceiptsRoot     []byte   `json:"receiptsRoot"`
	BaseFeePerGas    *big.Int `json:"baseFeePerGas"`
	Transactions     [][]byte `json:"transactions"`
	Uncles           [][]byte `json:"uncles"`
	TotalDifficulty  *big.Int `json:"totalDifficulty"`
}

//构造eth_block
func ConstructGetBlockResponseType(block *tpchaintypes.Block) *GetBlockResponseType {
	gasUsed := []byte(strconv.FormatUint(10_000_000_000, 16))
	transactionHashs := block.GetData().GetTxs()
	result := &GetBlockResponseType{
		Number:           block.GetHead().GetHeight(),
		Hash:             block.GetHead().GetHash(),
		ParentHash:       block.GetHead().GetParentBlockHash(),
		Nonce:            big.NewInt(0), //我们不用nonce，pow相关
		MixHash:          []byte{},      //也没有mixHash，pow相关
		Sha3Uncles:       []byte{},      //没有叔块
		LogsBloom:        []byte{},      //没有布隆过滤器
		StateRoot:        block.GetHead().GetStateRoot(),
		Miner:            block.GetHead().GetLauncher(),
		Difficulty:       big.NewInt(0), //没有难度
		ExtraData:        []byte{},      //没有额外数据
		Size:             block.Size(),
		GasLimit:         gasUsed,                      //整个区块的limit限制
		GasUsed:          block.GetHead().GetGasFees(), //整个区块的gas使用量
		Timestamp:        block.GetHead().GetTimeStamp(),
		TransactionsRoot: block.GetHead().GetTxRoot(),
		ReceiptsRoot:     block.GetHead().GetTxResultRoot(),
		BaseFeePerGas:    big.NewInt(0),    //没有的
		Transactions:     transactionHashs, //当前区块包含的所有交易的哈希
		Uncles:           [][]byte{},       //没有这个
		TotalDifficulty:  big.NewInt(0),    //没有这个
	}
	return result
}
