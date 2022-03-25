package consensus

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
)

func (m *PreparePackedMessageExe) TxsData() []byte {
	var txsData []byte
	for _, txData := range m.Txs {
		txsData = append(txsData, txData...)
	}

	return txsData
}

func (m *PreparePackedMessageProp) TxHashsData() []byte {
	var txHashsData []byte
	for _, txHashData := range m.TxHashs {
		txHashsData = append(txHashsData, txHashData...)
	}

	return txHashsData
}

func (m *ProposeMessage) BlockHeadInfo() (*tpchaintypes.BlockHead, error) {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	var bh tpchaintypes.BlockHead
	err := marshaler.Unmarshal(m.BlockHead, &bh)

	if err == nil {
		return &bh, nil
	}

	return nil, err
}
