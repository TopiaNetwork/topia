package secp256

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	PublicKeyBytes            = 65 //65 bytes
	PrivateKeyBytes           = 32 //32 bytes
	SignatureRecoverableBytes = 65 //65 bytes
	MsgBytes                  = 32 //32 bytes
	SeedBytes                 = 32 // 32 bytes
	maxLoopCreateSeckey       = 3
)

type CryptServiceSecp256 struct {
	log tplog.Logger
}

type asyEncryContent struct {
	Nonce      []byte              `json:"nonce"`
	Pubkey     tpcrtypes.PublicKey `json:"pubkey"`
	Ciphertext []byte              `json:"ciphertext"`
}

func New(log tplog.Logger) *CryptServiceSecp256 {
	return &CryptServiceSecp256{log}
}

func (c *CryptServiceSecp256) CryptType() tpcrtypes.CryptType {
	return tpcrtypes.CryptType_Secp256
}

var ctx = contextCreate()

func (c *CryptServiceSecp256) GeneratePriPubKey() (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	var seckey [PrivateKeyBytes]byte
	var pubkey [PublicKeyBytes]byte
	var randomize [32]byte

	if err := fillRandom(randomize[:]); err != nil {
		c.log.Error(err.Error())
		return nil, nil, err
	}

	if err := contextRandomize(ctx, randomize[:]); err != nil {
		c.log.Error(err.Error())
		return nil, nil, err
	}

	for i := 0; i < maxLoopCreateSeckey; i++ {
		if err := fillRandom(seckey[:]); err != nil {
			c.log.Error(err.Error())
			return nil, nil, err
		}
		verifyOK, err := ecSeckeyVerify(ctx, seckey[:])
		if err != nil {
			c.log.Error(err.Error())
			return nil, nil, err
		}
		if verifyOK == true {
			break
		}
	}

	pubkey, err := ecPubkeyCreateAndSerialize(ctx, seckey[:])
	if err != nil {
		c.log.Error(err.Error())
		return nil, nil, err
	}
	return seckey[:], pubkey[:], nil
}

func (c *CryptServiceSecp256) GeneratePriPubKeyBySeed(seed []byte) (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	if len(seed) != SeedBytes {
		return nil, nil, errors.New("seed length incorrect")
	}

	var seckey [PrivateKeyBytes]byte
	var pubkey [PublicKeyBytes]byte

	if err := contextRandomize(ctx, seed); err != nil {
		c.log.Error(err.Error())
		return nil, nil, err
	}

	for i := 0; i < maxLoopCreateSeckey; i++ {
		seckey = sha256.Sum256(seed)
		verifyOK, err := ecSeckeyVerify(ctx, seckey[:])
		if err != nil {
			c.log.Error(err.Error())
			return nil, nil, err
		}
		if verifyOK {
			break
		}
	}

	pubkey, err := ecPubkeyCreateAndSerialize(ctx, seckey[:])
	if err != nil {
		c.log.Error(err.Error())
		return nil, nil, err
	}
	return seckey[:], pubkey[:], nil
}

func (c *CryptServiceSecp256) ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error) {
	if len(priKey) != PrivateKeyBytes {
		return nil, errors.New("secp256 ConvertToPublic input seckey incorrect")
	}

	pubkey, err := ecPubkeyCreateAndSerialize(ctx, priKey)
	if err != nil {
		return nil, err
	}
	return pubkey[:], nil
}

func (c *CryptServiceSecp256) Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error) {
	signMsg := tpcmm.NewBlake2bHasher(MsgBytes).Compute(string(msg))
	if len(priKey) != PrivateKeyBytes || len(signMsg) != MsgBytes {
		return nil, errors.New("secp256 Sign input invalid parameter")
	}

	serializedSig, err := ecdsaSignAndSerialize(ctx, priKey, msg)
	if err != nil {
		return nil, err
	}
	return serializedSig[:], nil
}

func (c *CryptServiceSecp256) Verify(addr tpcrtypes.Address, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	if len(msg) != MsgBytes || signData == nil {
		return false, errors.New("secp256 Verify input invalid parameter")
	}

	pubKey, err := c.RecoverPublicKey(msg, signData)
	if err != nil {
		return false, err
	}

	retBool, err := ecdsaVerify(ctx, pubKey, signData, msg)
	if err != nil {
		return false, err
	}
	return retBool, nil
}

func (c *CryptServiceSecp256) RecoverPublicKey(msg []byte, signData tpcrtypes.Signature) (tpcrtypes.PublicKey, error) {
	if len(msg) != MsgBytes || len(signData) != SignatureRecoverableBytes {
		return nil, errors.New("input wrong parameter")
	}
	pubkeyArr, err := ecdsaRecoverPubkey(ctx, msg, signData)
	if err != nil {
		return nil, err
	}
	return pubkeyArr[:], nil
}

func (c *CryptServiceSecp256) StreamEncrypt(pubKey tpcrtypes.PublicKey, msg []byte) (encryptedData []byte, err error) {
	if len(pubKey) != PublicKeyBytes || msg == nil {
		return nil, errors.New("input invalid argument")
	}

	secIn, pubIn, err := c.GeneratePriPubKey()

	keySeed, err := ecPubkeyTweakMul(ctx, pubKey, secIn)
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

func (c *CryptServiceSecp256) StreamDecrypt(priKey tpcrtypes.PrivateKey, encryptedData []byte) (decryptedMsg []byte, err error) {
	if len(priKey) != PrivateKeyBytes || encryptedData == nil {
		return nil, errors.New("input invalid argument")
	}

	var receiver asyEncryContent
	err = json.Unmarshal(encryptedData, &receiver)
	if err != nil {
		return nil, err
	}

	keySeed, err := ecPubkeyTweakMul(ctx, receiver.Pubkey, priKey)
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

func (c *CryptServiceSecp256) CreateAddress(pubKey tpcrtypes.PublicKey) (tpcrtypes.Address, error) {
	addressHash := tpcmm.NewBlake2bHasher(tpcrtypes.AddressLen_Secp256).Compute(string(pubKey))
	if len(addressHash) != tpcrtypes.AddressLen_Secp256 {
		return tpcrtypes.UndefAddress, fmt.Errorf("Invalid addressHash: len %d, expected %d", len(addressHash), tpcrtypes.AddressLen_Secp256)
	}
	return tpcrtypes.NewAddress(tpcrtypes.CryptType_Secp256, addressHash)
}
