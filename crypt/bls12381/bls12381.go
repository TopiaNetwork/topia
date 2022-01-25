package bls12381

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type CryptServiceBLS12381 struct {
	log tplog.Logger
}

func New(log tplog.Logger) *CryptServiceBLS12381 {
	return &CryptServiceBLS12381{log}
}

func (c *CryptServiceBLS12381) GeneratePriPubKey() (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceBLS12381) ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceBLS12381) Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceBLS12381) Verify(pubKey tpcrtypes.PublicKey, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	//TODO implement me
	panic("implement me")
}
