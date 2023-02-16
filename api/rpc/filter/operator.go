package filter

type operator int

const (
	UnknownOperator operator = iota

	Eq // Eq -> "="

	Ne // Ne -> "!="

	Gt // Gt -> ">"

	Lt // Lt -> "<"

	Gte // Gte -> ">="

	Lte // Lte -> "<="

	Contains

	Exists
)

func (op *operator) isLegal() bool {
	_, ok := validOperator[*op]
	return ok
}

func (op *operator) string() string {
	return opToString[*op]
}

var reservedSigns = []string{">=", "<=", "!=", "=", ">", "<"}

var stringToOp = map[string]operator{
	"=":        Eq,
	"!=":       Ne,
	">":        Gt,
	"<":        Lt,
	">=":       Gte,
	"<=":       Lte,
	"contains": Contains,
	"exists":   Exists,
}

var opToString = map[operator]string{
	Eq:       "=",
	Ne:       "!=",
	Gt:       ">",
	Lt:       "<",
	Gte:      ">=",
	Lte:      "<=",
	Contains: "contains",
	Exists:   "exists",
}

var validOperator = map[operator]struct{}{
	Eq:       {},
	Ne:       {},
	Gt:       {},
	Lt:       {},
	Gte:      {},
	Lte:      {},
	Contains: {},
	Exists:   {},
}
