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

	// Valid returns whether the current iterator is valid. Once invalid, the Iterator remains
	// invalid forever.
	Valid() bool

	// Next moves the iterator to the next key in the database, as defined by order of iteration.
	// If Valid returns false, this method will panic.
	Next()

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
