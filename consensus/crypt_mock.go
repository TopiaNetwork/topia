package consensus

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type CryptServiceMock struct{}

func (cs *CryptServiceMock) GeneratePriPubKey() (tpcrtypes.PrivateKey, tpcrtypes.PublicKey, error) {
	pubKey, priKey, err := ed25519.GenerateKey(rand.Reader)

	return tpcrtypes.PrivateKey(priKey), tpcrtypes.PublicKey(pubKey), err
}

func (cs *CryptServiceMock) ConvertToPublic(priKey tpcrtypes.PrivateKey) (tpcrtypes.PublicKey, error) {
	edPriKey := ed25519.PrivateKey(priKey)

	switch pubType := edPriKey.Public().(type) {
	case ed25519.PublicKey:
		return tpcrtypes.PublicKey(pubType), nil
	}

	return nil, fmt.Errorf("invalid private key: %v", priKey)
}

func (cs *CryptServiceMock) Sign(priKey tpcrtypes.PrivateKey, msg []byte) (tpcrtypes.Signature, error) {
	edPriKey := ed25519.PrivateKey(priKey)

	signData := ed25519.Sign(edPriKey, msg)
	if signData == nil {
		return nil, fmt.Errorf("Invalid sign: private key=%v, msg=%v", priKey, msg)
	}

	return signData, nil
}

func (cs *CryptServiceMock) Verify(pubKey tpcrtypes.PublicKey, msg []byte, signData tpcrtypes.Signature) (bool, error) {
	edPubKey := ed25519.PublicKey(pubKey)

	return ed25519.Verify(edPubKey, msg, signData), nil
}
