package types

import types "github.com/TopiaNetwork/topia/api/web3/types/hexutil"

type GetBlockByHashRequestType struct {
	BlockHash string
	Tx        bool
}

type GetBlockResponseType struct {
	Number           *types.Big    `json:"number"`
	Hash             Hash          `json:"hash"`
	ParentHash       Hash          `json:"parentHash"`
	Nonce            []byte        `json:"nonce"`
	MixHash          Hash          `json:"mixHash"`
	Sha3Uncles       Hash          `json:"sha3Uncles"`
	LogsBloom        []byte        `json:"logsBloom"`
	StateRoot        Hash          `json:"stateRoot"`
	Miner            Address       `json:"miner"`
	Difficulty       *types.Big    `json:"difficulty"`
	ExtraData        []byte        `json:"extraData"`
	Size             types.Uint64  `json:"size"`
	GasLimit         types.Uint64  `json:"gasLimit"`
	GasUsed          types.Uint64  `json:"gasUsed"`
	Timestamp        types.Uint64  `json:"timestamp"`
	TransactionsRoot Hash          `json:"transactionsRoot"`
	ReceiptsRoot     Hash          `json:"receiptsRoot"`
	BaseFeePerGas    *types.Big    `json:"baseFeePerGas"`
	Transactions     []interface{} `json:"transactions"`
	Uncles           []Hash        `json:"uncles"`
	TotalDifficulty  *types.Big    `json:"totalDifficulty"`
}
