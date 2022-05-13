package types

type GetBalanceRequestType struct {
	Address string
	Height  interface{}
}

type GetBalanceResponseType struct {
	Balance string `json:"balance"`
}
