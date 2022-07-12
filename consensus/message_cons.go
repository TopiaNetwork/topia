package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/lazyledger/smt"
)

func (m *ConsensusDomainSelectedMessage) DataBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, m.MemberNumber)

	var dataBytes []byte
	dataBytes = append(dataBytes, m.DomainID...)
	dataBytes = append(dataBytes, b...)
	dataBytes = append(dataBytes, m.NodeIDOfMember...)

	return dataBytes
}

func (m *PreparePackedMessageExe) TxsData() []byte {
	var txsData []byte
	for _, txData := range m.Txs {
		txsData = append(txsData, txData...)
	}

	return txsData
}

func (m *PreparePackedMessageExeIndication) DataBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, m.StateVersion)

	return b
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

func (m *ProposeMessage) TxRoot() []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, txHashBytes := range m.TxHashs {
		tree.Update(txHashBytes, txHashBytes)
	}

	return tree.Root()
}

func (m *ProposeMessage) TxResultRoot() []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, txRSHashBytes := range m.TxResultHashs {
		tree.Update(txRSHashBytes, txRSHashBytes)
	}

	return tree.Root()
}

func (m *ExeResultValidateReqMessage) TxAndResultHashsData() []byte {
	var hashsData []byte
	for _, txHashData := range m.TxHashs {
		hashsData = append(hashsData, txHashData...)
	}
	for _, txRSData := range m.TxResultHashs {
		hashsData = append(hashsData, txRSData...)
	}

	return hashsData
}

func (m *ExeResultValidateRespMessage) TxAndResultProofsData() []byte {
	var proofsData []byte
	for _, txProofData := range m.TxProofs {
		proofsData = append(proofsData, txProofData...)
	}
	for _, txRSProofData := range m.TxResultProofs {
		proofsData = append(proofsData, txRSProofData...)
	}

	return proofsData
}
