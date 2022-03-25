package common

// Batch represents a group of writes. They may or may not be written atomically depending on the
// backend. Callers must call Close on the batch when done.
//
// As with DB, given keys and values should be considered read-only, and must not be modified after
// passing them to the batch.
type Batch interface {
	// Set sets a key/value pair.
	// CONTRACT: key, value readonly []byte
	Set(key, value []byte) error

	// Delete deletes a key/value pair.
	// CONTRACT: key readonly []byte
	Delete(key []byte) error

	// Write writes the batch, possibly without flushing to disk. Only Close() can be called after,
	// other methods will error.
	Write() error

	// WriteSync writes the batch and flushes it to disk. Only Close() can be called after, other
	// methods will error.
	WriteSync() error

	// Close closes the batch. It is idempotent, but calls to other methods afterwards will error.
	Close() error
}

// Iterator represents an iterator over a domain of keys. Callers must call Close when done.
type Iterator interface {
	// Domain returns the start (inclusive) and end (exclusive) limits of the iterator.
	// CONTRACT: start, end readonly []byte
	Domain() (start []byte, end []byte)

	// Next moves the iterator to the next key in the database, as defined by order of iteration;
	// returns whether the iterator is valid.
	// Once this function returns false, the iterator remains invalid forever.
	Next() bool

	// Key returns the key at the current position. Panics if the iterator is invalid.
	// CONTRACT: key readonly []byte
	Key() (key []byte)

	// Value returns the value at the current position. Panics if the iterator is invalid.
	// CONTRACT: value readonly []byte
	Value() (value []byte)

	// Error returns the last error encountered by the iterator, if any.
	Error() error

	// Close closes the iterator, relasing any allocated resources.
	Close() error
}

// VersionSet specifies a set of existing versions
type VersionSet interface {
	// Last returns the most recent saved version, or 0 if none.
	Last() uint64
	// Count returns the number of saved versions.
	Count() int
	// Iterator returns an iterator over all saved versions.
	Iterator() VersionIterator
	// Equal returns true iff this set is identical to another.
	Equal(VersionSet) bool
	// Exists returns true if a saved version exists.
	Exists(uint64) bool
}

type VersionIterator interface {
	// Next advances the iterator to the next element.
	// Returns whether the iterator is valid; once invalid, it remains invalid forever.
	Next() bool
	// Value returns the version ID at the current position.
	Value() uint64
}

// DBReader is a read-only transaction interface. It is safe for concurrent access.
// Callers must call Discard when done with the transaction.
//
// Keys cannot be nil or empty, while values cannot be nil. Keys and values should be considered
// read-only, both when returned and when given, and must be copied before they are modified.
type DBReader interface {
	// Get fetches the value of the given key, or nil if it does not exist.
	// CONTRACT: key, value readonly []byte
	Get([]byte) ([]byte, error)

	// Has checks if a key exists.
	// CONTRACT: key, value readonly []byte
	Has(key []byte) (bool, error)

	// Iterator returns an iterator over a domain of keys, in ascending order. The caller must call
	// Close when done. End is exclusive, and start must be less than end. A nil start iterates
	// from the first key, and a nil end iterates to the last key (inclusive). Empty keys are not
	// valid.
	// CONTRACT: No writes may happen within a domain while an iterator exists over it.
	// CONTRACT: start, end readonly []byte
	Iterator(start, end []byte) (Iterator, error)

	// ReverseIterator returns an iterator over a domain of keys, in descending order. The caller
	// must call Close when done. End is exclusive, and start must be less than end. A nil end
	// iterates from the last key (inclusive), and a nil start iterates to the first key (inclusive).
	// Empty keys are not valid.
	// CONTRACT: No writes may happen within a domain while an iterator exists over it.
	// CONTRACT: start, end readonly []byte
	// TODO: replace with an extra argument to Iterator()?
	ReverseIterator(start, end []byte) (Iterator, error)

	// Discard discards the transaction, invalidating any future operations on it.
	Discard() error
}

// DBWriter is a write-only transaction interface.
// It is safe for concurrent writes, following an optimistic (OCC) strategy, detecting any write
// conflicts and returning an error on commit, rather than locking the DB.
// Callers must call Commit or Discard when done with the transaction.
//
// This can be used to wrap a write-optimized batch object if provided by the backend implementation.
type DBWriter interface {
	// Set sets the value for the given key, replacing it if it already exists.
	// CONTRACT: key, value readonly []byte
	Set([]byte, []byte) error

	// Delete deletes the key, or does nothing if the key does not exist.
	// CONTRACT: key readonly []byte
	Delete([]byte) error

	// Commit flushes pending writes and discards the transaction.
	Commit() error

	// Discard discards the transaction, invalidating any future operations on it.
	Discard() error
}

// DBReadWriter is a transaction interface that allows both reading and writing.
type DBReadWriter interface {
	DBReader
	DBWriter
}
