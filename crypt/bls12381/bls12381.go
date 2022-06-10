package bls12381

import (
	"errors"
	"fmt"
	"github.com/TopiaNetwork/go-bls"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	PublicKeyBytes  = 48 //48 bytes
	PrivateKeyBytes = 32 //32 bytes
)

const MCL_BLS12_381 = bls.MCL_BLS12_381

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
	bls.Initialization(MCL_BLS12_381)
	sec := bls.CreateSecretKey()
	pub := sec.GetPublicKey()
	if sec == nil || pub == nil {
		errStr := "GeneratePriPubKey priKey or pubKey is nil"
		c.log.Error(errStr)
		return nil, nil, errors.New(errStr)
	}
	secRet := sec.Serialize()
	pubRet := pub.Serialize()
	if len(secRet) != PrivateKeyBytes || len(pubRet) != PublicKeyBytes {
		errStr := "GeneratePriPubKey key length incorrect"
		c.log.Error(errStr)
		return nil, nil, errors.New(errStr)
	}
	return secRet, pubRet, nil
}

func (c *CryptServiceBLS12381) GeneratePriPubKeyWithSeed(seed []byte) (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CryptServiceBLS12381) ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error) {
	var sec bls.SecretKey
	if priKey == nil {
		return nil, errors.New("ConvertToPublic: input PrivateKey err")
	}
	if err := sec.Deserialize(priKey); err != nil {
		return nil, err
	}
	return sec.GetPublicKey().Serialize(), nil
}

func (c *CryptServiceBLS12381) Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error) {
	var sec bls.SecretKey
	if priKey == nil || msg == nil {
		return nil, errors.New("Sign: input PrivateKey or msg err")
	}
	if err := sec.Deserialize(priKey); err != nil {
		return nil, err
	}
	return sec.Sign(string(msg)).Serialize(), nil
}

func (c *CryptServiceBLS12381) Verify(pubKey tpcrtypes.PublicKey, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	if pubKey == nil || msg == nil || signData == nil {
		return false, errors.New("Verify: input err")
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
