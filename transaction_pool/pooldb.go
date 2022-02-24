package transactionpool

import (
	"github.com/TopiaNetwork/topia/account"
	"math/big"
)

type PoolDB interface {
	GetAccount(addr account.Address) (*account.Account, error)
}
type StatePoolDB struct {
	Acc *account.Account
}

func newStatePoolDB() (*StatePoolDB,error)  {
	var stateDb = &StatePoolDB{
		&account.Account{
			"",
			"",
			0,
			big.NewInt(0),
		},
	}
	return stateDb,nil
}


func (s *StatePoolDB) GetAccount(addr account.Address)(*account.Account, error){
	return s.Acc,nil
}
func (s *StatePoolDB) GetBalance(addr account.Address) *big.Int {

	return s.Acc.Balance
}
func (s *StatePoolDB) GetNonce(addr account.Address) uint64{
	return s.Acc.Nonce
}
