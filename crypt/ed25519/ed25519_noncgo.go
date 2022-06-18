//go:build !cgo
// +build !cgo

package ed25519

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	PublicKeyBytes  = ed25519.PublicKeySize  // 32 bytes
	PrivateKeyBytes = ed25519.PrivateKeySize // 64 bytes
	SignatureBytes  = ed25519.SignatureSize  // 64 bytes
	KeyGenSeedBytes = ed25519.SeedSize       // 32 bytes
)

type CryptServiceEd25519 struct {
	log tplog.Logger
}

func New(log tplog.Logger) *CryptServiceEd25519 {
	return &CryptServiceEd25519{log}
}

func (c *CryptServiceEd25519) CryptType() tpcrtypes.CryptType {
	return tpcrtypes.CryptType_Ed25519
}

func (c *CryptServiceEd25519) GeneratePriPubKey() (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	pub, sec, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}
	return []byte(sec), []byte(pub), nil
}

func (c *CryptServiceEd25519) GeneratePriPubKeyBySeed(seed []byte) (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	if len(seed) != KeyGenSeedBytes {
		return nil, nil, errors.New("input seed length err")
	}
	sec := ed25519.NewKeyFromSeed(seed)
	pub := make([]byte, PublicKeyBytes)
	copy(pub, sec[32:])
	return []byte(sec), pub, nil
}

func (c *CryptServiceEd25519) ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error) {
	if len(priKey) != PrivateKeyBytes {
		return nil, errors.New("input invalid PrivateKey")
	}
	pub := make([]byte, PublicKeyBytes)
	copy(pub, priKey[32:])
	return pub, nil
}

func (c *CryptServiceEd25519) Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error) {
	if len(priKey) != PrivateKeyBytes || len(msg) == 0 {
		return nil, errors.New("input invalid argument")
	}
	sig := ed25519.Sign([]byte(priKey), msg)
	return sig, nil
}

func (c *CryptServiceEd25519) Verify(addr tpcrtypes.Address, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	if len(pubKey) != PublicKeyBytes || len(msg) == 0 || len(signData) != SignatureBytes {
		return false, errors.New("input invalid argument")
	}

	pubKey, err := pubKeyFromAddr(addr)
	if err != nil {
		return false, err
	}

	return ed25519.Verify([]byte(pubKey), msg, signData), nil
}

func (c *CryptServiceEd25519) BatchVerify(pubKeys []tpcrtypes.PublicKey, msgs [][]byte, signDatas []tpcrtypes.Signature) (bool, error) {
	if len(pubKeys) != len(signDatas) || len(pubKeys) != len(msgs) {
		return false, errors.New("input invalid argument")
	}
	for i := range pubKeys {
		if len(pubKeys[i]) != PublicKeyBytes || len(msgs[i]) == 0 || len(signDatas[i]) != SignatureBytes {
			return false, errors.New("input invalid argument")
		}
	}

	for i := range pubKeys {
		retBool := ed25519.Verify([]byte(pubKeys[i]), msgs[i], signDatas[i])
		if retBool == false {
			return false, nil
		}
	}
	return true, nil
}

func (c *CryptServiceEd25519) CreateAddress(pubKey tpcrtypes.PublicKey) (tpcrtypes.Address, error) {
	if len(pubKey) != PublicKeyBytes {
		return tpcrtypes.UndefAddress, fmt.Errorf("Invalid pubKey: len %d, expected %d", len(pubKey), tpcrtypes.AddressLen_ED25519)
	}
	return tpcrtypes.NewAddress(tpcrtypes.CryptType_Ed25519, pubKey)
}
