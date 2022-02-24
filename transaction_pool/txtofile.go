package transactionpool

import (
"errors"
"github.com/TopiaNetwork/topia/account"
"github.com/TopiaNetwork/topia/transaction"
"github.com/ethereum/go-ethereum/rlp"
"io"
"os"
)
var errNoActiveJournal = errors.New("no active journal")

// devNull is a WriteCloser that just discards anything written into it. Its
// goal is to allow the transaction journal to write into a fake journal when
// loading transactions on startup without printing warnings due to no file
// being read for write.
type devNull struct{}

func (*devNull) Write(p []byte) (n int, err error) { return len(p), nil }
func (*devNull) Close() error                      { return nil }

// txStored is to save and load local or remote transactions with.
type txStored struct {
	pathLocal	string         // Filesystem path to store the local transactions at
	pathRemote	string         // Filesystem path to store the remote transactions at
	writer	io.WriteCloser     // Output stream to write new transactions into
}

func newTxStored(pathLocal,pathRemote string) *txStored {
	return &txStored{
		pathLocal:	pathLocal,
		pathRemote:	pathRemote,
	}
}

func (stored *txStored) loadLocal(add func([]*transaction.Transaction) []error) error {
	// Skip the parsing if the journal file doesn't exist at all
	if _, err := os.Stat(stored.pathLocal); os.IsNotExist(err) {
		return nil
	}
	// Open the journal for loading any past transactions
	input, err := os.Open(stored.pathLocal)
	if err != nil {
		return err
	}
	defer input.Close()

	stored.writer = new(devNull)
	defer func() { stored.writer = nil }()

	stream := rlp.NewStream(input, 0)
	total, dropped := 0, 0

	loadBatch := func(txs []*transaction.Transaction) {
		for _, err := range add(txs) {
			if err != nil {
				//log.Debug("Failed to add journaled transaction", "err", err
				dropped++
			}
		}
	}
	var (
		failure error
		batch   []*transaction.Transaction
	)
	for {
		// Parse the next transaction and terminate on error
		tx := new(transaction.Transaction)
		if err = stream.Decode(tx); err != nil {
			if err != io.EOF {
				failure = err
			}
			if len(batch) > 0 {
				loadBatch(batch)
			}
			break
		}
		// New transaction parsed, queue up for later, import if threshold is reached
		total++

		if batch = append(batch, tx); len(batch) > 1024 {
			loadBatch(batch)
			batch = batch[:0]
		}
	}
	//log.Info("Loaded local transaction journal", "transactions", total, "dropped", dropped)
	return failure
}
func (stored *txStored) loadRemote(add func([]*transaction.Transaction) []error) error {
	if _, err := os.Stat(stored.pathRemote); os.IsNotExist(err) {
		return nil
	}
	input, err := os.Open(stored.pathRemote)
	if err != nil {
		return err
	}
	defer input.Close()

	stored.writer = new(devNull)
	defer func() { stored.writer = nil }()

	stream := rlp.NewStream(input, 0)
	total, dropped := 0, 0

	loadBatch := func(txs []*transaction.Transaction) {
		for _, err := range add(txs) {
			if err != nil {
				//log.Debug("Failed to add stored transaction", "err", err)
				dropped++
			}
		}
	}
	var (
		failure error
		batch   []*transaction.Transaction
	)
	for {
		tx := new(transaction.Transaction)
		if err = stream.Decode(tx); err != nil {
			if err != io.EOF {
				failure = err
			}
			if len(batch) > 0 {
				loadBatch(batch)
			}
			break
		}
		// New transaction parsed, queue up for later, import if threshold is reached
		total++

		if batch = append(batch, tx); len(batch) > 1024 {
			loadBatch(batch)
			batch = batch[:0]
		}
	}
	//log.Info("Loaded local transaction journal", "transactions", total, "dropped", dropped)
	return failure
}

func (stored *txStored) insert(tx *transaction.Transaction) error {
	if stored.writer == nil {
		return errNoActiveJournal
	}
	if err := rlp.Encode(stored.writer, tx); err != nil {
		return err
	}
	return nil
}

// save regenerates the transaction journal based on the current contents of
// the transaction pool.
func (stored *txStored) saveLocal(all map[account.Address][]*transaction.Transaction) error {
	// Close the current journal (if any is open)
	if stored.writer != nil {
		if err := stored.writer.Close(); err != nil {
			return err
		}
		stored.writer = nil
	}
	replacement, err := os.OpenFile(stored.pathLocal+".new", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	journaled := 0
	for _, txs := range all {
		for _, tx := range txs {
			if err = rlp.Encode(replacement, tx); err != nil {
				replacement.Close()
				return err
			}
		}
		journaled += len(txs)
	}
	replacement.Close()
	// Replace the live journal with the newly generated one
	if err = os.Rename(stored.pathLocal+".new", stored.pathLocal); err != nil {
		return err
	}
	sink, err := os.OpenFile(stored.pathLocal, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	stored.writer = sink
	//log.Info("Regenerated local transaction journal", "transactions", journaled, "accounts", len(all))
	//no achieved for log
	return nil
}

// rotate regenerates the transaction journal based on the current contents of
// the transaction pool.
func (stored *txStored) saveRemote(all map[account.Address][]*transaction.Transaction) error {
	if stored.writer != nil {
		if err := stored.writer.Close(); err != nil {
			return err
		}
		stored.writer = nil
	}
	// Generate a new journal with the contents of the current pool
	replacement, err := os.OpenFile(stored.pathRemote+".new", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	journaled := 0
	for _, txs := range all {
		for _, tx := range txs {
			if err = rlp.Encode(replacement, tx); err != nil {
				replacement.Close()
				return err
			}
		}
		journaled += len(txs)
	}
	replacement.Close()
	// Replace the live journal with the newly generated one
	if err = os.Rename(stored.pathRemote+".new", stored.pathRemote); err != nil {
		return err
	}
	sink, err := os.OpenFile(stored.pathRemote, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	stored.writer = sink
	//log.Info("Regenerated local transaction journal", "transactions", journaled, "accounts", len(all))
	//no achieved for log
	return nil
}

// close flushes the transaction journal contents to disk and closes the file.
func (stored *txStored) close() error {
	var err error

	if stored.writer != nil {
		err = stored.writer.Close()
		stored.writer = nil
	}
	return err
}

