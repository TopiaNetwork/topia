package common

type LedgerState byte

const (
	LedgerState_Uninitialized LedgerState = iota
	LedgerState_Genesis
	LedgerState_AutoInc
)
