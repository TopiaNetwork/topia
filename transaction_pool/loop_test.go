package transactionpool

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func Test_transactionPool_loop_chanRemoveTxHashes(t *testing.T) {
	pool1.TruncateTxPool()

	var hashes1, hashes2 []txbasic.TxID
	var hash txbasic.TxID

	for _, tx := range txLocals[:10] {
		hash, _ = tx.TxID()
		hashes1 = append(hashes1, hash)
		pool1.AddTx(tx, false)
	}

	for _, tx := range txLocals[20:40] {
		hash, _ = tx.TxID()
		hashes2 = append(hashes2, hash)
		pool1.AddTx(tx, false)
	}
	for _, tx := range pool1.GetRemoteTxs() {
		fmt.Println("addr,nonce", tx.Head.FromAddr, tx.Head.Nonce)
	}
	assert.Equal(t, 27, len(pool1.GetRemoteTxs()))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool1.RemoveTxBatch(hashes1)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool1.RemoveTxBatch(hashes2)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(8 * time.Second)
		pool1.ctx.Done()
	}()

	wg.Wait()
	for _, tx := range pool1.GetRemoteTxs() {
		fmt.Println("addr,nonce", tx.Head.FromAddr, tx.Head.Nonce)
	}
	assert.Equal(t, 0, len(pool1.GetRemoteTxs()))
	pool1.TruncateTxPool()

}

func Test_transactionPool_loop_saveAllIfShutDown(t *testing.T) {
	pool1.TruncateTxPool()

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)

	pool1.SysShutDown()
	time.Sleep(5 * time.Second)
	fmt.Println("done")

}

func Test_transactionPool_loopDropTxsIfBlockAdded(t *testing.T) {
	pool1.TruncateTxPool()

	txs := txLocals[10:40]

	pool1.addTxs(txs, true)
	assert.Equal(t, 27, len(pool1.GetLocalTxs()))
	assert.Equal(t, 0, len(pool1.GetRemoteTxs()))
	//OldBlock txs:txLocals[10:20]，remotes[10:20]
	pool1.chanBlockAdded <- OldBlock
	time.Sleep(10 * time.Second)
	assert.Equal(t, 18, len(pool1.GetLocalTxs()))
	assert.Equal(t, 0, len(pool1.GetRemoteTxs()))

}

func Test_transactionPool_removeTxForUptoLifeTime(t *testing.T) {
	pool1.TruncateTxPool()

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)

	//**********for test
	//*change default lifTime to 4 second,and change TxExpiredTime to 3 second **
	//**********for test
	pool1.config.LifetimeForTx = 4 * time.Second
	assert.Equal(t, int64(4), pool1.Count())
	time.Sleep(13 * time.Second)
	locals := pool1.GetLocalTxs()
	if len(locals) > 0 {
		for _, tx := range locals {
			fmt.Println(tx.Head.FromAddr, tx.Head.Nonce)
		}
	}
	remotes := pool1.GetRemoteTxs()
	if len(remotes) > 0 {
		for _, tx := range remotes {
			fmt.Println(tx.Head.FromAddr, tx.Head.Nonce)
		}
	}
	assert.Equal(t, int64(0), pool1.Count())
	fmt.Println("test down")

}

func Test_transactionPool_regularSaveLocalTxs(t *testing.T) {
	pool1.TruncateTxPool()

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)
	//**********for test
	//*change default ReStoredDur to 4 second **
	//**********for test
	time.Sleep(10 * time.Second)
	pool1.ClearLocalFile(pool1.config.PathTxsStorage)
	//you can see this log:
	// loadTxsData file removed  module=TransactionPool
}

func Test_transactionPool_regularRepublic(t *testing.T) {
	pool1.TruncateTxPool()

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)

	wrapT1, ok := pool1.allWrappedTxs.Get(Key1)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, wrapT1.IsRepublished)
	wrapT2, ok := pool1.allWrappedTxs.Get(KeyR2)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, wrapT2.IsRepublished)

	//**********for test
	//*change RepublishTxInterval to 3000 * time.Millisecond,
	// change default TxTTLTimeOfRepublish to 4 *time.Second
	//**********for test
	time.Sleep(13 * time.Second)

	wrapT1, ok = pool1.allWrappedTxs.Get(Key1)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, wrapT1.IsRepublished)
	wrapT2, ok = pool1.allWrappedTxs.Get(Key2)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, wrapT2.IsRepublished)
	fmt.Println("test down")

}

func Test_transactionPool_loop(t *testing.T) {
	pool1.TruncateTxPool()

	keyLocals = make([]txbasic.TxID, 0)
	keyRemotes = make([]txbasic.TxID, 0)
	txLocals = make([]*txbasic.Transaction, 0)
	txRemotes = make([]*txbasic.Transaction, 0)
	for i := 1; i <= 100; i++ {
		nonce := uint64(i)
		gasprice := uint64(i * 1000)
		gaslimit := uint64(i * 1000000)

		txlocal = setTxLocal(nonce, gasprice, gaslimit)
		txlocal.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txlocal.Head.FromAddr = append(txlocal.Head.FromAddr, byte(i))
		}
		keylocal, _ = txlocal.TxID()
		keyLocals = append(keyLocals, keylocal)
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.TxID()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)

		pool1.AddTx(txlocal, true)
		pool1.AddTx(txremote, false)
	}
	assert.Equal(t, int64(200), pool1.Count())
	go func() {
		pool1.RemoveTxBatch(keyLocals)
	}()
	go func() {
		pool1.RemoveTxBatch(keyRemotes)
	}()

	pool1.chanBlockAdded <- NewBlock
	pool1.SysShutDown()

	time.Sleep(20 * time.Second)
	assert.Equal(t, int64(0), pool1.Count())

}
