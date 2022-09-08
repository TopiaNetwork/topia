package store

type Store interface {
	AddNamedStateStore(name string, cacheSize int) error

	Root(name string) ([]byte, error)

	Put(name string, key []byte, value []byte) error

	Delete(name string, key []byte) error

	Exists(name string, key []byte) (bool, error)

	Update(name string, key []byte, value []byte) error

	Clone(other Store) error

	Commit() error

	Rollback() error

	Stop() error

	Close() error
}
