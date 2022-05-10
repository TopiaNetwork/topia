package eventhub

import txbasic "github.com/TopiaNetwork/topia/transaction/basic"

type TxPoolEVType byte

const (
	TxPoolEVType_Unknown TxPoolEVType = iota
	TxPoolEVType_Received
	TxPoolEVTypee_Removed
)

type TxPoolEvent struct {
	evType TxPoolEVType
	tx     *txbasic.Transaction
}

func (te TxPoolEVType) String() string {
	switch te {
	case TxPoolEVType_Received:
		return "TxPoolEVType_Received"
	case TxPoolEVTypee_Removed:
		return "TxPoolEVType_Removed"
	default:
		return "TxPoolEVType_Unknown"
	}
}

func (te TxPoolEVType) Value(evName string) TxPoolEVType {
	switch evName {
	case "TxPoolEVType_Received":
		return TxPoolEVType_Received
	case "TxPoolEVType_Removed":
		return TxPoolEVTypee_Removed
	default:
		return TxPoolEVType_Unknown
	}
}
