package types

type GetBalanceRequestType struct {
	Address string
	Height  string
}

type GetBalanceResponseType struct {
	Balance string `json:"balance"`
}
