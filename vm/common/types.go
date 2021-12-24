package common

type VMInputParas struct {
	FromAddr   []byte
	TargetAddr []byte
	Function   string
	Arguments  [][]byte
}

type VMResult struct {
	ReturnData [][]byte
}
