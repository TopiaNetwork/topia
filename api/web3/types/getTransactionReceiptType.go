package types

type GetTransactionReceiptRequestType struct {
	TxHash Hash
}

type GetTransactionReceiptResponseType struct {
	BlockHash         string      `json:"blockHash"`
	BlockNumber       string      `json:"blockNumber"`
	TransactionHash   string      `json:"transactionHash"`
	TransactionIndex  string      `json:"transactionIndex"`
	From              string      `json:"from"`
	To                interface{} `json:"to"`
	GasUsed           string      `json:"gasUsed"`
	CumulativeGasUsed string      `json:"cumulativeGasUsed"`
	ContractAddress   string      `json:"contractAddress"`
	Logs              string      `json:"logs"`
	LogsBloom         string      `json:"logsBloom"`
	Type              string      `json:"type"`
	EffectiveGasPrice string      `json:"effectiveGasPrice"`
	Status            string      `json:"status"`
}
