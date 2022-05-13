package convert

import (
	tpTypes "github.com/TopiaNetwork/topia/api/web3/types"
	types "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"math/big"
)

func ConvertTopiaBlockToEthBlock(block *tpchaintypes.Block) *tpTypes.GetBlockResponseType {
	transactionHashs := block.GetData().GetTxs()
	transactionHashString := make([]interface{}, 0)
	for _, v := range transactionHashs {
		transactionHashString = append(transactionHashString, string(v))
	}

	gas := big.NewInt(10_000_000_000)
	height := new(big.Int).SetUint64(block.GetHead().GetHeight())
	var blockHash tpTypes.Hash
	blockHash.SetBytes(block.GetHead().GetHash())
	var parentBlockHash tpTypes.Hash
	parentBlockHash.SetBytes(block.GetHead().GetHash())
	var stateRoot tpTypes.Hash
	stateRoot.SetBytes(block.GetHead().GetStateRoot())
	var miner tpTypes.Address
	miner.SetBytes(block.GetHead().GetLauncher())
	gasUsed := types.Uint64(new(big.Int).SetBytes(block.GetHead().GetGasFees()).Uint64())
	var txToor tpTypes.Hash
	txToor.SetBytes(block.GetHead().GetTxRoot())
	var txReceiptRoot tpTypes.Hash
	txReceiptRoot.SetBytes(block.GetHead().GetTxResultRoot())
	result := &tpTypes.GetBlockResponseType{
		Number:           (*types.Big)(height),
		Hash:             blockHash,
		ParentHash:       parentBlockHash,
		Nonce:            []byte{},
		StateRoot:        stateRoot,
		Miner:            miner,
		Size:             types.Uint64(block.Size()),
		GasLimit:         types.Uint64(gas.Uint64()),
		GasUsed:          gasUsed,
		Timestamp:        types.Uint64(block.GetHead().GetTimeStamp()),
		TransactionsRoot: txToor,
		ReceiptsRoot:     txReceiptRoot,
		Transactions:     transactionHashString,
	}
	return result
}
