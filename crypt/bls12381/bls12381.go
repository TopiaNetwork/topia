package bls12381

import (
	"fmt"
	bls "github.com/TopiaNetwork/go-bls"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
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
	bls.Initialization(bls.MCL_BLS12_381)
	sec := bls.CreateSecretKey()
	pub := sec.GetPublicKey()
	if sec == nil || pub == nil {
		err := fmt.Errorf("GeneratePriPubKey err\n")
		return nil, nil, err
	}
	return sec.Serialize(), pub.Serialize(), nil
}

func (c *CryptServiceBLS12381) ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error) {
	var sec bls.SecretKey
	if priKey == nil {
		return nil, fmt.Errorf("ConvertToPublic: input PrivateKey err\n")
	}
	if err := sec.Deserialize(priKey); err != nil {
		return nil, err
	}
	return sec.GetPublicKey().Serialize(), nil
}

func (c *CryptServiceBLS12381) Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error) {
	var sec bls.SecretKey
	if priKey == nil || msg == nil {
		return nil, fmt.Errorf("Sign: input PrivateKey or msg err\n")
	}
	if err := sec.Deserialize(priKey); err != nil {
		return nil, err
	}
	return sec.Sign(string(msg)).Serialize(), nil
}

func (c *CryptServiceBLS12381) Verify(pubKey tpcrtypes.PublicKey, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	if pubKey == nil || msg == nil || signData == nil {
		return false, fmt.Errorf("Verify: input err\n")
	}
	var pub bls.PublicKey
	var sig bls.Signature
	if err := pub.Deserialize(pubKey); err != nil {
		return false, err
	}
	if err := sig.Deserialize(signData); err != nil {
		return false, err
	}
	return sig.Verify(&pub, string(msg)), nil
}

func (c *CryptServiceBLS12381) CreateAddress(pubKey tpcrtypes.PublicKey) (tpcrtypes.Address, error) {
	if len(pubKey) != tpcrtypes.AddressLen_BLS12381 {
		return tpcrtypes.UndefAddress, fmt.Errorf("Invalid pubkey: len %d, expected %d", len(pubKey), tpcrtypes.AddressLen_BLS12381)
	}
	return tpcrtypes.NewAddress(tpcrtypes.CryptType_BLS12381, pubKey)
}
