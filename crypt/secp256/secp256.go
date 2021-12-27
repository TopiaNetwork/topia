package secp256

import (
	"github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type CryptServiceSecp256 struct {
	log tplog.Logger
}

func New(log tplog.Logger) *CryptServiceSecp256 {
	return &CryptServiceSecp256{log}
}

func (c *CryptServiceSecp256) GeneratePriPubKey() (types.PrivateKey, types.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceSecp256) ConvertToPublic() (types.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceSecp256) Sign(priKey types.PrivateKey, msg []byte) (types.Signature, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceSecp256) Verify(pubKey types.PublicKey, signData types.Signature) error {
	//TODO implement me
	panic("implement me")
}
