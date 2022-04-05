package transactionpool

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type remoteTxs struct {
	Txs                 map[string]*basic.Transaction
	ActivationIntervals map[string]time.Time
}

func (pool *transactionPool) SaveRemoteTxs() error {

	var remotetxs = &remoteTxs{}
	remotetxs.Txs = pool.allTxsForLook.remotes
	remotetxs.ActivationIntervals = pool.ActivationIntervals.activ

	remotes, err := json.Marshal(remotetxs)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(pool.config.PathRemote, remotes, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) loadRemote(nofile bool, path string) {
	if !nofile && path != "" {
		if err := pool.LoadRemoteTxs(); err != nil {
			pool.log.Warnf("Failed to load remote transactions", "err", err)
		}
	}
}
func (pool *transactionPool) LoadRemoteTxs() error {
	data, err := ioutil.ReadFile(pool.config.PathRemote)
	if err != nil {
		return nil
	}
	remotetxs := &remoteTxs{}
	err = json.Unmarshal(data, &remotetxs)
	if err != nil {
		return nil
	}

	for _, tx := range remotetxs.Txs {
		pool.AddRemote(tx)
	}
	for txId, ActivationInterval := range remotetxs.ActivationIntervals {
		pool.ActivationIntervals.activ[txId] = ActivationInterval
	}
	return nil
}

// AddRemotes enqueues a batch of transactions into the pool if they are valid. If the
// senders are not among the locally tracked ones, full pricing constraints will apply.
//
// This method is used to add transactions from the p2p network and does not wait for pool
// reorganization and internal event propagation.
func (pool *transactionPool) AddRemotes(txs []*basic.Transaction) []error {
	return pool.addTxs(txs, false, false)
}
func (pool *transactionPool) AddRemote(tx *basic.Transaction) error {
	errs := pool.AddRemotes([]*basic.Transaction{tx})

	return errs[0]
}

//dispatch
func (pool *transactionPool) Dispatch(context actor.Context, data []byte) {
	var txPoolMsg TxPoolMessage
	err := pool.marshaler.Unmarshal(data, &txPoolMsg)
	if err != nil {
		pool.log.Errorf("TransactionPool receive invalid data %v", data)
		return
	}

	switch txPoolMsg.MsgType {
	case TxPoolMessage_Tx:
		var msg TxMessage
		err := pool.marshaler.Unmarshal(txPoolMsg.Data, &msg)
		if err != nil {
			pool.log.Errorf("TransactionPool unmarshal msg %d err %v", txPoolMsg.MsgType, err)
			return
		}
		err = pool.processTx(&msg)
		if err != nil {
			return
		}
	default:
		pool.log.Errorf("TransactionPool receive invalid msg %d", txPoolMsg.MsgType)
		return
	}
}

func (pool *transactionPool) processTx(msg *TxMessage) error {
	err := pool.handler.ProcessTx(msg)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) BroadCastTx(tx *basic.Transaction) error {
	if tx == nil {
		return nil
	}
	msg := &TxMessage{}
	data, err := tx.GetData().Marshal()
	if err != nil {
		return err
	}
	msg.Data = data
	_, err = msg.Marshal()
	if err != nil {
		return err
	}
	var toModuleName string
	toModuleName = MOD_NAME
	pool.network.Publish(pool.ctx, toModuleName, protocol.SyncProtocolID_Msg, data)
	return nil
}
