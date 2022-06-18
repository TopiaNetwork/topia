package ed25519

import (
	"crypto/rand"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGeneratePriPubKey(t *testing.T) {
	var c CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "private key length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "public key length err")
	assert.Equal(t, nil, err, "GeneratePriPubKey err")
}

func TestGeneratePriPubKeyBySeed(t *testing.T) {
	var c CryptServiceEd25519
	seed := make([]byte, KeyGenSeedBytes)
	_, err := rand.Read(seed)
	assert.Equal(t, nil, err, "generate rand num err")
	sec, pub, err := c.GeneratePriPubKeyBySeed(seed)
	assert.Equal(t, PrivateKeyBytes, len(sec), "private key length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "public key length err")
	assert.Equal(t, nil, err, "GeneratePriPubKey err")
}

func TestConvertToPublic(t *testing.T) {
	var c CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "private key length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "public key length err")
	assert.Equal(t, nil, err, "GeneratePriPubKey err")

	pubConvert, err := c.ConvertToPublic(sec)
	assert.Equal(t, nil, err, "ConvertToPublic err")
	for i := range pub {
		assert.Equal(t, pub[i], pubConvert[i], "ConvertToPublic err")
	}
}

func TestSign(t *testing.T) {
	var c CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "private key length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "public key length err")
	assert.Equal(t, nil, err, "GeneratePriPubKey err")

	msg := []byte("this is test msg for sign")
	sig, err := c.Sign(sec, msg)
	assert.Equal(t, SignatureBytes, len(sig), "signature length err")
	assert.Equal(t, nil, err, "Sign err")
}

func TestVerify(t *testing.T) {
	var c CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "private key length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "public key length err")
	assert.Equal(t, nil, err, "GeneratePriPubKey err")

	msg := []byte("this is test msg for sign")
	sig, err := c.Sign(sec, msg)
	assert.Equal(t, SignatureBytes, len(sig), "signature length err")
	assert.Equal(t, nil, err, "Sign err")

	addr, err := c.CreateAddress(pub)
	assert.Nil(t, err, "CreateAddress err")

	retBool, err := c.Verify(addr, msg, sig)
	assert.Equal(t, true, retBool, "Verify err")
	assert.Equal(t, nil, err, "Verify err")
}

func TestBatchVerify(t *testing.T) {
	var c CryptServiceEd25519
	num := 10
	secs := make([]tpcrtypes.PrivateKey, num)
	pubs := make([]tpcrtypes.PublicKey, num)
	msgs := make([][]byte, num)
	sigs := make([]tpcrtypes.Signature, num)
	addrs := make([]tpcrtypes.Address, num)
	var err error

	for i := 0; i < num; i++ {
		secs[i], pubs[i], err = c.GeneratePriPubKey()
		assert.Equal(t, PrivateKeyBytes, len(secs[i]), "private key length err")
		assert.Equal(t, PublicKeyBytes, len(pubs[i]), "public key length err")
		assert.Equal(t, nil, err, "GeneratePriPubKey err")
	}
	msgLen := 50
	for i := range msgs {
		msgs[i] = make([]byte, msgLen)
		_, err = rand.Read(msgs[i])
		assert.Equal(t, nil, err, "generate random msg err")
	}
	for i := range msgs {
		sigs[i], err = c.Sign(secs[i], msgs[i])
		assert.Equal(t, SignatureBytes, len(sigs[i]), "signature length err")
		assert.Equal(t, nil, err, "Sign err")
	}

	for i := range addrs {
		tempAddr, err := c.CreateAddress(pubs[i])
		assert.Nil(t, err, "CreateAddress err")
		addrs[i] = tempAddr
	}

	retBool, err := c.BatchVerify(addrs, msgs, sigs)
	assert.Equal(t, true, retBool, "BatchVerify err")
	assert.Equal(t, nil, err, "BatchVerify err")
}

func TestCreateAddress(t *testing.T) {
	var c CryptServiceEd25519
	_, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, nil, err, "GeneratePriPubKey err")
	_, err = c.CreateAddress(pub)
	assert.Equal(t, nil, err, "CreateAddress err")
}
