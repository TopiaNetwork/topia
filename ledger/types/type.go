package types

type QueryResult interface{}

type ResultsIterator interface {
	Next() (QueryResult, error)
	Close()
}
