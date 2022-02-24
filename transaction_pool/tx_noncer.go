package transactionpool

import (
	"github.com/TopiaNetwork/topia/account"
	"sync"
)

type txNoncer struct {
	fallback   *StatePoolDB
	nonces     map[account.Address]uint64
	lock       sync.Mutex
}

// newTxNoncer creates a new virtual state database to track the pool nonces.
func newTxNoncer(statedb *StatePoolDB) *txNoncer {
	return &txNoncer{
		fallback: statedb,
		nonces:   make(map[account.Address]uint64),
	}
}

// get returns the current nonce of an account, falling back to a real state
// database if the account is unknown.
func (txn *txNoncer) get(addr account.Address) uint64 {
	// We use mutex for get operation is the underlying
	// state will mutate db even for read access.
	txn.lock.Lock()
	defer txn.lock.Unlock()

	if _, ok := txn.nonces[addr]; !ok {
		txn.nonces[addr] = txn.fallback.GetNonce(addr)
	}
	return txn.nonces[addr]
}

// set inserts a new virtual nonce into the virtual state database to be returned
// whenever the pool requests it instead of reaching into the real state database.
func (txn *txNoncer) set(addr account.Address, nonce uint64) {
	txn.lock.Lock()
	defer txn.lock.Unlock()

	txn.nonces[addr] = nonce
}

// setIfLower updates a new virtual nonce into the virtual state database if
// the new one is lower.
func (txn *txNoncer) setIfLower(addr account.Address, nonce uint64) {
	txn.lock.Lock()
	defer txn.lock.Unlock()

	if _, ok := txn.nonces[addr]; !ok {
		txn.nonces[addr] = txn.fallback.GetNonce(addr)
	}
	if txn.nonces[addr] <= nonce {
		return
	}
	txn.nonces[addr] = nonce
}

// setAll sets the nonces for all accounts to the given map.
func (txn *txNoncer) setAll(all map[account.Address]uint64) {
	txn.lock.Lock()
	defer txn.lock.Unlock()
	txn.nonces = all
}
