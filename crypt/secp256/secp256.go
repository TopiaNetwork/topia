package secp256

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type CryptServiceSecp256 struct {
	log tplog.Logger
}

func New(log tplog.Logger) *CryptServiceSecp256 {
	return &CryptServiceSecp256{log}
}

func (c *CryptServiceSecp256) GeneratePriPubKey() (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceSecp256) ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceSecp256) Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceSecp256) Verify(pubKey tpcrtypes.PublicKey, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	//TODO implement me
	panic("implement me")
}
