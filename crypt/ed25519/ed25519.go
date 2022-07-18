//go:build cgo
// +build cgo

package ed25519

import "C"
import (
	"crypto/rand"
	"errors"
	"fmt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	PublicKeyBytes            = 32 // 32 bytes
	PrivateKeyBytes           = 64 // 64 bytes
	SignatureBytes            = 64 // 64 bytes
	KeyGenSeedBytes           = 32 // 32 bytes
	Curve25519PublicKeyBytes  = 32 // 32 bytes
	Curve25519PrivateKeyBytes = 32 // 32 bytes
)

type CryptServiceEd25519 struct {
	log tplog.Logger
}

func New(log tplog.Logger) *CryptServiceEd25519 {
	return &CryptServiceEd25519{log}
}

func pubKeyFromAddr(addr tpcrtypes.Address) (tpcrtypes.PublicKey, error) {
	payload, err := addr.Payload()
	if err != nil {
		return nil, err
	}
	if len(payload) != PublicKeyBytes {
		return nil, fmt.Errorf("Expecte verifying pubKey %d, actual %d", PublicKeyBytes, len(payload))
	}

	return payload, nil
}

func (c *CryptServiceEd25519) CryptType() tpcrtypes.CryptType {
	return tpcrtypes.CryptType_Ed25519
}

func (c *CryptServiceEd25519) GeneratePriPubKey() (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	seed := make([]byte, KeyGenSeedBytes)
	if _, err := rand.Read(seed); err != nil {
		return nil, nil, errors.New("fill random seed err")
	}

	return generateKeyPairFromSeed(seed)
}

func (c *CryptServiceEd25519) GeneratePriPubKeyBySeed(seed []byte) (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	if len(seed) != KeyGenSeedBytes {
		return nil, nil, errors.New("input seed length err")
	}

	return generateKeyPairFromSeed(seed)
}

func (c *CryptServiceEd25519) ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error) {
	if len(priKey) != PrivateKeyBytes {
		return nil, errors.New("input invalid privateKey")
	}

	return seckeyToPubkey(priKey)
}

func (c *CryptServiceEd25519) Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error) {
	if len(priKey) != PrivateKeyBytes || len(msg) == 0 {
		return nil, errors.New("input invalid argument")
	}

	return signDetached(priKey, msg)
}

func (c *CryptServiceEd25519) Verify(addr tpcrtypes.Address, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	if len(msg) == 0 || len(signData) != SignatureBytes {
		return false, errors.New("input invalid argument")
	}

	pubKey, err := pubKeyFromAddr(addr)
	if err != nil {
		return false, err
	}

	return verifyDetached(pubKey, msg, signData)
}

func (c *CryptServiceEd25519) BatchVerify(addrs []tpcrtypes.Address, msgs [][]byte, signDatas []tpcrtypes.Signature) (bool, error) {
	if len(addrs) != len(signDatas) || len(addrs) != len(msgs) {
		return false, errors.New("input invalid argument")
	}
	for i := range addrs {
		if len(msgs[i]) == 0 || len(signDatas[i]) != SignatureBytes {
			return false, errors.New("input invalid argument")
		}
	}

	pubKeys := make([]tpcrtypes.PublicKey, len(addrs))
	for i := range pubKeys {
		tempPubkey, err := pubKeyFromAddr(addrs[i])
		if err != nil {
			return false, err
		}
		pubKeys[i] = tempPubkey
	}

	return batchVerify(pubKeys, msgs, signDatas), nil
}

func (c *CryptServiceEd25519) CreateAddress(pubKey tpcrtypes.PublicKey) (tpcrtypes.Address, error) {
	if len(pubKey) != PublicKeyBytes {
		return tpcrtypes.UndefAddress, fmt.Errorf("Invalid pubKey: len %d, expected %d", len(pubKey), PublicKeyBytes)
	}
	return tpcrtypes.NewAddress(tpcrtypes.CryptType_Ed25519, pubKey)
}

func (c *CryptServiceEd25519) batchVerifyOneByOne(addrs []tpcrtypes.Address, msgs [][]byte, signDatas []tpcrtypes.Signature) (bool, error) {
	if len(addrs) != len(signDatas) || len(addrs) != len(msgs) {
		return false, errors.New("input invalid argument")
	}
	for i := range addrs {
		if len(msgs[i]) == 0 || len(signDatas[i]) != SignatureBytes {
			return false, errors.New("input invalid argument")
		}
	}

	pubKeys := make([]tpcrtypes.PublicKey, len(addrs))
	for i := range pubKeys {
		tempPubkey, err := pubKeyFromAddr(addrs[i])
		if err != nil {
			return false, err
		}
		pubKeys[i] = tempPubkey
		_, err = verifyDetached(pubKeys[i], msgs[i], signDatas[i])
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func ToCurve25519(sec tpcrtypes.PrivateKey, pub tpcrtypes.PublicKey) (curveSec []byte, curvePub []byte, err error) {
	if len(sec) != PrivateKeyBytes || len(pub) != PublicKeyBytes {
		return nil, nil, errors.New("input invalid argument")
	}
	curveSec, curvePub, err = toCurve25519(sec, pub)
	return curveSec, curvePub, err
}

func (c *CryptServiceEd25519) StreamEncrypt(pubKey tpcrtypes.PublicKey, msg []byte) (encryptedData []byte, err error) {
	if len(msg) == 0 {
		return nil, errors.New("input invalid argument")
	}
	return streamEncrypt(pubKey, msg)
}

func (c *CryptServiceEd25519) StreamDecrypt(priKey tpcrtypes.PrivateKey, encryptedData []byte) (decryptedMsg []byte, err error) {
	if len(encryptedData) == 0 {
		return nil, errors.New("input invalid argument")
	}
	return streamDecrypt(priKey, encryptedData)
}
