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

func TestStreamEncryptDecrypt(t *testing.T) {
	password := []byte("this is password1")
	msg := []byte("this is the msg to steam encrypt")
	encryptedMsg, err := StreamEncrypt(password, msg)
	assert.Nil(t, err, "StreamEncrypt err:", err)

	decryptedMsg, err := StreamDecrypt(password, encryptedMsg)
	assert.Nil(t, err, "StreamDecrypt err:", err)
	assert.Equal(t, string(msg), string(decryptedMsg), "StreamDecrypt failed")

	password2 := []byte("this is password2")
	msg2 := []byte("this is the 2 msg to steam encrypt")
	encryptedMsg2, err := StreamEncrypt(password2, msg2)
	assert.Nil(t, err, "StreamEncrypt err:", err)

	decryptedMsg2, err := StreamDecrypt(password2, encryptedMsg2)
	assert.Nil(t, err, "StreamDecrypt err:", err)
	assert.Equal(t, string(msg2), string(decryptedMsg2), "StreamDecrypt failed")
}
