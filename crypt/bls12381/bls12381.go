package bls12381

import (
	"fmt"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	AddressLen      = 48 //48 bytes
	PublicKeyBytes  = 48 //48 bytes
	PrivateKeyBytes = 32 //32 bytes
)

type CryptServiceBLS12381 struct {
	log tplog.Logger
}

func New(log tplog.Logger) *CryptServiceBLS12381 {
	return &CryptServiceBLS12381{log}
}

func (c *CryptServiceBLS12381) CryptType() tpcrtypes.CryptType {
	return tpcrtypes.CryptType_BLS12381
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

func (c *CryptServiceBLS12381) CreateAddress(pubKey tpcrtypes.PublicKey) (tpcrtypes.Address, error) {
	if len(pubKey) != AddressLen {
		return tpcrtypes.UndefAddress, fmt.Errorf("Invalid pubkey: len %d, expected %d", len(pubKey), AddressLen)
	}
	return tpcrtypes.NewAddress(tpcrtypes.CryptType_BLS12381, pubKey)
}
