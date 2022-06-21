package crypt

import (
	"github.com/TopiaNetwork/topia/crypt/ed25519"
	"github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type CryptService interface {
	CryptType() tpcrtypes.CryptType

	GeneratePriPubKey() (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error)

	GeneratePriPubKeyWithSeed(seed []byte) (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error)

	ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error)

	Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error)

	Verify(addr tpcrtypes.Address, msg []byte, signData tpcrtypes.Signature) (bool, error)

	CreateAddress(pubKey tpcrtypes.PublicKey) (tpcrtypes.Address, error)
}

func CreateCryptService(log tplog.Logger, cryptType tpcrtypes.CryptType) CryptService {
	cryptLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "crypt", log)
	switch cryptType {
	case tpcrtypes.CryptType_Ed25519:
		return ed25519.New(cryptLog)
	case tpcrtypes.CryptType_Secp256:
		return secp256.New(cryptLog)
	default:
		cryptLog.Panicf("invalid crypt type %d", cryptType)
	}

	return nil
}
