package transaction

type TxID string

type TransactionType byte

const (
	TransactionType_Unknown TransactionType = iota
	TransactionType_Transfer
	TransactionType_ContractDeploy
	TransactionType_ContractInvoke
	TransactionType_NativeInvoke
	TransactionType_Pay
	TransactionType_Relay
	TransactionType_DataTransfer
)

func (m *Transaction) GetType() TransactionType {
	panic("implement me")
}
