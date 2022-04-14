package currency

const TPAPrecision = uint64(1_000_000_000_000_000_000)

type TokenSymbol string

const (
	TokenSymbol_UnKnown TokenSymbol = ""
	TokenSymbol_Native              = "TPA"
)
