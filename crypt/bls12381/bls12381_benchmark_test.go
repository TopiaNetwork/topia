package bls12381

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkSign(b *testing.B) {
	b.StopTimer()
	var c CryptServiceBLS12381
	sec, _, err := c.GeneratePriPubKey()
	assert.Equal(b, nil, err, "BenchmarkSign GeneratePriPubKey failed.")
	msgToSign := "This is the msg for sign."
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		_, err = c.Sign(sec, []byte(msgToSign))
		b.StopTimer()
		assert.Equal(b, nil, err, "BenchmarkSign Sign failed.")

	}
}

func BenchmarkVerify(b *testing.B) {
	b.StopTimer()
	var c CryptServiceBLS12381
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(b, nil, err, "BenchmarkVerify GeneratePriPubKey failed.")

	msgToSign := "This is the msg for sign."
	sig, err := c.Sign(sec, []byte(msgToSign))
	assert.Equal(b, nil, err, "BenchmarkVerify Sign failed.")

	addr, err := c.CreateAddress(pub)
	assert.Nil(b, err, "TestVerify CreateAddress failed.")

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		bo, err := c.Verify(addr, []byte(msgToSign), sig)
		b.StopTimer()
		assert.Equal(b, true, bo, "BenchmarkVerify Verify failed.")
		assert.Equal(b, nil, err, "BenchmarkVerify Verify failed.")
	}

}
