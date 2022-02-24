package transaction



import (
"github.com/TopiaNetwork/topia/account"
"math/big"
)


type sigCache struct {
	signer Signer
	from   account.Address
}
type Signer interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *Transaction) (account.Address, error)

	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
	ChainID() *big.Int

	// Hash returns 'signature hash', i.e. the transaction hash that is signed by the
	// private key. This hash does not uniquely identify the transaction.
	Hash(tx *Transaction) account.Address

	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}

type BaseSigner struct {
	chainId,chainIdMul *big.Int
}
func NewBaseSigner(chainId *big.Int) BaseSigner {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return BaseSigner{
		chainId: chainId,
		chainIdMul: new(big.Int).Mul(chainId,big.NewInt(2)),
	}
}

func (s BaseSigner) ChainID() *big.Int {
	return s.chainId
}

func (s BaseSigner) Equal(s2 Signer) bool {
	eip155, ok := s2.(Signer)
	return ok && eip155.ChainID().Cmp(s.chainId) == 0
}

var big8 = big.NewInt(8)

func Sender(signer BaseSigner,tx *Transaction) (account.Address, error) {
	//no achived for signer
	if sc := tx.FromAddr[:];sc != nil {
		_ = signer
		return account.Address(tx.FromAddr[:]),nil
	}
	return account.Address(tx.FromAddr[:]),nil
}
func MakeSigner()BaseSigner{
	var signer BaseSigner
	return signer
}

