//go:build cgo
// +build cgo

package ed25519

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const benchCgoGroupAmount10 = 10
const benchCgoGroupAmount50 = 50
const benchCgoGroupAmount100 = 100

func TestEd25519ToCurve25519(t *testing.T) {
	var c CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "private key length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "public key length err")
	assert.Equal(t, nil, err, "GeneratePriPubKey err")

	curveSec, curvePub, err := ToCurve25519(sec, pub)
	assert.Equal(t, Curve25519PrivateKeyBytes, len(curveSec), "private key convert to curve25519 length err")
	assert.Equal(t, Curve25519PublicKeyBytes, len(curvePub), "public key convert to curve25519 length err")
	assert.Equal(t, nil, err, "convert to curve25519 err")
}

func BenchmarkBatchVerifyOneByOne10(b *testing.B) {
	b.StopTimer()
	var c CryptServiceEd25519
	addrs, msgs, sigs := prepareForBatchVerify(benchCgoGroupAmount10)
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		retBool, err := c.batchVerifyOneByOne(addrs, msgs, sigs)
		b.StopTimer()
		assert.Equal(b, true, retBool, "BatchVerify err")
		assert.Equal(b, nil, err, "BatchVerify err")
	}
}
func BenchmarkBatchVerifyOneByOne50(b *testing.B) {
	b.StopTimer()
	var c CryptServiceEd25519
	addrs, msgs, sigs := prepareForBatchVerify(benchCgoGroupAmount50)
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		retBool, err := c.batchVerifyOneByOne(addrs, msgs, sigs)
		b.StopTimer()
		assert.Equal(b, true, retBool, "BatchVerify err")
		assert.Equal(b, nil, err, "BatchVerify err")
	}
}
func BenchmarkBatchVerifyOneByOne100(b *testing.B) {
	b.StopTimer()
	var c CryptServiceEd25519
	addrs, msgs, sigs := prepareForBatchVerify(benchCgoGroupAmount100)
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		retBool, err := c.batchVerifyOneByOne(addrs, msgs, sigs)
		b.StopTimer()
		assert.Equal(b, true, retBool, "BatchVerify err")
		assert.Equal(b, nil, err, "BatchVerify err")
	}
}
