package crypt

import (
	"github.com/TopiaNetwork/topia/crypt/bls12381"
	"github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type CryptServiceType int

const (
	CryptServiceType_Unknown CryptServiceType = iota
	CryptServiceType_BLS12381
	CryptServiceType_Secp256
)

type CryptService interface {
	GeneratePriPubKey() (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error)

	ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error)

	Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error)

	Verify(pubKey tpcrtypes.PublicKey, msg []byte, signData tpcrtypes.Signature) (bool, error)
}

func CreateCryptService(log tplog.Logger, cryptType CryptServiceType) CryptService {
	cryptLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "crypt", log)
	switch cryptType {
	case CryptServiceType_BLS12381:
		return bls12381.New(cryptLog)
	case CryptServiceType_Secp256:
		return secp256.New(cryptLog)
	default:
		cryptLog.Panicf("invalid crypt type %d", cryptType)
	}

	return nil
}
