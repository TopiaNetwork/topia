package eventhub

import txbasic "github.com/TopiaNetwork/topia/transaction/basic"

type TxPoolEvent_Type byte

const (
	TxPoolEvent_Type_Received TxPoolEvent_Type = iota + 1
	TxPoolEvent_Type_Removed
)

type TxPoolEvent struct {
	kind TxPoolEvent_Type
	tx   *txbasic.Transaction
}
