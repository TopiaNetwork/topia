package bls12381

import (
	"github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type CryptServiceBLS12381 struct {
	log tplog.Logger
}

func New(log tplog.Logger) *CryptServiceBLS12381 {
	return &CryptServiceBLS12381{log}
}

func (c *CryptServiceBLS12381) GeneratePriPubKey() (types.PrivateKey, types.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceBLS12381) ConvertToPublic() (types.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceBLS12381) Sign(priKey types.PrivateKey, msg []byte) (types.Signature, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceBLS12381) Verify(pubKey types.PublicKey, signData types.Signature) error {
	//TODO implement me
	panic("implement me")
}
