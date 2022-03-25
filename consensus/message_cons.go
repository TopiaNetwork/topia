package consensus

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
