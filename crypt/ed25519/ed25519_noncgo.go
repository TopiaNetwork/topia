//go:build !cgo
// +build !cgo

package ed25519

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	"golang.org/x/crypto/curve25519"
	"math/big"
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

type asyEncryContent struct {
	Nonce      []byte              `json:"nonce"`
	Pubkey     tpcrtypes.PublicKey `json:"pubkey"`
	Ciphertext []byte              `json:"ciphertext"`
}

var curve25519P, _ = new(big.Int).SetString("57896044618658097711785492504343953926634992332820282019728792003956564819949", 10)

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

	return ed25519.Sign([]byte(priKey), msg), nil
}

func (c *CryptServiceEd25519) Verify(addr tpcrtypes.Address, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	if len(msg) == 0 || len(signData) != SignatureBytes {
		return false, errors.New("input invalid argument")
	}

	pubKey, err := pubKeyFromAddr(addr)
	if err != nil {
		return false, err
	}

	return ed25519.Verify([]byte(pubKey), msg, signData), nil
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
		ok := ed25519.Verify([]byte(pubKeys[i]), msgs[i], signDatas[i])
		if !ok {
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

func (c *CryptServiceEd25519) StreamEncrypt(pubKey tpcrtypes.PublicKey, msg []byte) (encryptedData []byte, err error) {
	if len(pubKey) != PublicKeyBytes || msg == nil {
		return nil, errors.New("input invalid argument")
	}

	secIn, pubIn, err := c.GeneratePriPubKey()

	secCurveIn := ed25519PrivateKeyToCurve25519(ed25519.PrivateKey(secIn))
	pubCurve := ed25519PublicKeyToCurve25519(ed25519.PublicKey(pubKey))

	keySeed, err := curve25519.X25519(secCurveIn, pubCurve)
	if err != nil {
		return nil, err
	}

	key := sha256.Sum256(keySeed)

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, msg, nil)

	return json.Marshal(asyEncryContent{
		Nonce:      nonce,
		Pubkey:     pubIn,
		Ciphertext: ciphertext,
	})
}

func (c *CryptServiceEd25519) StreamDecrypt(priKey tpcrtypes.PrivateKey, encryptedData []byte) (decryptedMsg []byte, err error) {
	if len(priKey) != PrivateKeyBytes || encryptedData == nil {
		return nil, errors.New("input invalid argument")
	}

	var receiver asyEncryContent
	err = json.Unmarshal(encryptedData, &receiver)
	if err != nil {
		return nil, err
	}

	secCurve := ed25519PrivateKeyToCurve25519(ed25519.PrivateKey(priKey))
	pubCurveIn := ed25519PublicKeyToCurve25519(ed25519.PublicKey(receiver.Pubkey))

	keySeed, err := curve25519.X25519(secCurve, pubCurveIn)
	if err != nil {
		return nil, err
	}

	key := sha256.Sum256(keySeed)

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesgcm.Open(nil, receiver.Nonce, receiver.Ciphertext, nil)
}

func ed25519PublicKeyToCurve25519(pubkey ed25519.PublicKey) []byte {
	bigEndianY := make([]byte, ed25519.PublicKeySize)
	for i, b := range pubkey {
		bigEndianY[ed25519.PublicKeySize-i-1] = b
	}
	bigEndianY[0] &= 0b0111_1111

	y := new(big.Int).SetBytes(bigEndianY)
	denom := big.NewInt(1)
	denom.ModInverse(denom.Sub(denom, y), curve25519P)
	u := y.Mul(y.Add(y, big.NewInt(1)), denom)
	u.Mod(u, curve25519P)

	out := make([]byte, curve25519.PointSize)
	uBytes := u.Bytes()
	for i, b := range uBytes {
		out[len(uBytes)-i-1] = b
	}

	return out
}

func ed25519PrivateKeyToCurve25519(seckey ed25519.PrivateKey) []byte {
	h := sha512.New()
	h.Write(seckey.Seed())
	out := h.Sum(nil)

	out[0] &= 248
	out[31] &= 127
	out[31] |= 64

	return out[:curve25519.ScalarSize]
}
