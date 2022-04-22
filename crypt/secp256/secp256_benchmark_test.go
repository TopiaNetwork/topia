package secp256

import (
	"crypto/sha256"
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkSign(b *testing.B) {
	b.StopTimer()
	var c CryptServiceSecp256
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(b, PrivateKeyBytes, len(sec), "Generate seckey length err")
	assert.Equal(b, PublicKeyBytes, len(pub), "Generate pubkey length err")
	assert.Equal(b, nil, err, "Generate key err")

	msgArr := sha256.Sum256([]byte("This is the message for sign"))
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		_, err = c.Sign(sec, msgArr[:])
		b.StopTimer()
		assert.Equal(b, nil, err, "Sign err")
	}
}

func BenchmarkVerify(b *testing.B) {
	b.StopTimer()
	var c CryptServiceSecp256
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(b, PrivateKeyBytes, len(sec), "Generate seckey length err")
	assert.Equal(b, PublicKeyBytes, len(pub), "Generate pubkey length err")
	assert.Equal(b, nil, err, "Generate key err")

	msgArr := sha256.Sum256([]byte("This is the message for sign"))
	sig, err := c.Sign(sec, msgArr[:])
	assert.Equal(b, nil, err, "Sign err")

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		retBool, err := c.Verify(pub, msgArr[:], sig)
		b.StopTimer()
		assert.Equal(b, true, retBool, "Verify should be true")
		assert.Equal(b, nil, err, "Verify err")
	}
}
