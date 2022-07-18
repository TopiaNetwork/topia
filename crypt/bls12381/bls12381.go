package bls12381

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
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

type asyEncryContent struct {
	Nonce      []byte              `json:"nonce"`
	Pubkey     tpcrtypes.PublicKey `json:"pubkey"`
	Ciphertext []byte              `json:"ciphertext"`
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

func (c *CryptServiceBLS12381) GeneratePriPubKeyBySeed(noNeedSeed []byte) (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	return c.GeneratePriPubKey()
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

func (c *CryptServiceBLS12381) Verify(addr tpcrtypes.Address, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	if msg == nil || signData == nil {
		return false, errors.New("Verify: input err")
	}
	var pub bls.PublicKey
	var sig bls.Signature

	pubKey, err := pubKeyFromAddr(addr)
	if err != nil {
		return false, err
	}

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
func (c *CryptServiceBLS12381) StreamEncrypt(pubKey tpcrtypes.PublicKey, msg []byte) (encryptedData []byte, err error) {
	if len(pubKey) != PublicKeyBytes || msg == nil {
		return nil, errors.New("input invalid argument")
	}

	bls.Initialization(MCL_BLS12_381)
	secIn := bls.CreateSecretKey()
	pubIn := secIn.GetPublicKey()

	keyGenerator := new(bls.PublicKey)
	err = keyGenerator.Deserialize(pubKey)
	if err != nil {
		return nil, err
	}
	keyGenerator.Mul(secIn)
	key := sha256.Sum256(keyGenerator.Serialize())

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
		Pubkey:     pubIn.Serialize(),
		Ciphertext: ciphertext,
	})
}

func (c *CryptServiceBLS12381) StreamDecrypt(priKey tpcrtypes.PrivateKey, encryptedData []byte) (decryptedMsg []byte, err error) {
	if len(priKey) != PrivateKeyBytes || encryptedData == nil {
		return nil, errors.New("input invalid argument")
	}

	var receiver asyEncryContent
	err = json.Unmarshal(encryptedData, &receiver)
	if err != nil {
		return nil, err
	}

	keyGenerator := new(bls.PublicKey)
	err = keyGenerator.Deserialize(receiver.Pubkey)
	if err != nil {
		return nil, err
	}

	tempSec := new(bls.SecretKey)
	err = tempSec.Deserialize(priKey)
	if err != nil {
		return nil, err
	}
	keyGenerator.Mul(tempSec)
	key := sha256.Sum256(keyGenerator.Serialize())

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
