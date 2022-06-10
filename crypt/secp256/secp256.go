package secp256

import (
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
	maxLoopCreateSeckey       = 3
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

func (c *CryptServiceSecp256) GeneratePriPubKeyWithSeed(seed []byte) (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	//TODO implement me
	panic("implement me")
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
	if len(priKey) != PrivateKeyBytes || len(msg) != MsgBytes {
		return nil, errors.New("secp256 Sign input invalid parameter")
	}

	serializedSig, err := ecdsaSignAndSerialize(ctx, priKey, msg)
	if err != nil {
		return nil, err
	}
	return serializedSig[:], nil
}

func (c *CryptServiceSecp256) Verify(pubKey tpcrtypes.PublicKey, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	if len(pubKey) != PublicKeyBytes || len(msg) != MsgBytes || signData == nil {
		return false, errors.New("secp256 Verify input invalid parameter")
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

func (c *CryptServiceSecp256) CreateAddress(pubKey tpcrtypes.PublicKey) (tpcrtypes.Address, error) {
	addressHash := tpcmm.NewBlake2bHasher(tpcrtypes.AddressLen_Secp256).Compute(string(pubKey))
	if len(addressHash) != tpcrtypes.AddressLen_Secp256 {
		return tpcrtypes.UndefAddress, fmt.Errorf("Invalid addressHash: len %d, expected %d", len(addressHash), tpcrtypes.AddressLen_Secp256)
	}
	return tpcrtypes.NewAddress(tpcrtypes.CryptType_Secp256, addressHash)
}
