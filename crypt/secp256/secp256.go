package secp256

import (
	"fmt"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	PublicKeyBytes  = 65 //65 bytes
	PrivateKeyBytes = 32 //32 bytes
)

type CryptServiceSecp256 struct {
	log tplog.Logger
}

func New(log tplog.Logger) *CryptServiceSecp256 {
	return &CryptServiceSecp256{log}
}

func (c *CryptServiceSecp256) CryptType() tpcrtypes.CryptType {
	return tpcrtypes.CryptType_Secp256
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

func (c *CryptServiceSecp256) CreateAddress(pubKey tpcrtypes.PublicKey) (tpcrtypes.Address, error) {
	addressHash := tpcmm.NewBlake2bHasher(tpcrtypes.AddressLen_Secp256).Compute(string(pubKey))
	if len(addressHash) != tpcrtypes.AddressLen_Secp256 {
		return tpcrtypes.UndefAddress, fmt.Errorf("Invalid addressHash: len %d, expected %d", len(addressHash), tpcrtypes.AddressLen_Secp256)
	}
	return tpcrtypes.NewAddress(tpcrtypes.CryptType_Secp256, addressHash)
}
