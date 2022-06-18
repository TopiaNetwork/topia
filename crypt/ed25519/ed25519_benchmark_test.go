package ed25519

import (
	"crypto/rand"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

const benchGroupAmount10 = 10
const benchGroupAmount50 = 50
const benchGroupAmount100 = 100

func BenchmarkSign(b *testing.B) {
	b.StopTimer()
	var c CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(b, PrivateKeyBytes, len(sec), "private key length err")
	assert.Equal(b, PublicKeyBytes, len(pub), "public key length err")
	assert.Equal(b, nil, err, "GeneratePriPubKey err")

	msg := []byte("this is test msg for sign")
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		sig, err := c.Sign(sec, msg)
		b.StopTimer()
		assert.Equal(b, SignatureBytes, len(sig), "signature length err")
		assert.Equal(b, nil, err, "Sign err")
	}
}

func BenchmarkVerify(b *testing.B) {
	b.StopTimer()
	var c CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(b, PrivateKeyBytes, len(sec), "private key length err")
	assert.Equal(b, PublicKeyBytes, len(pub), "public key length err")
	assert.Equal(b, nil, err, "GeneratePriPubKey err")

	addr, err := c.CreateAddress(pub)
	assert.Nil(b, err, "CreateAddress err")

	msg := []byte("this is test msg for sign")
	sig, err := c.Sign(sec, msg)
	assert.Equal(b, SignatureBytes, len(sig), "signature length err")
	assert.Equal(b, nil, err, "Sign err")

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		retBool, err := c.Verify(addr, msg, sig)
		b.StopTimer()
		assert.Equal(b, true, retBool, "Verify err")
		assert.Equal(b, nil, err, "Verify err")
	}
}

func prepareForBatchVerify(groupNum int) (addrs []tpcrtypes.Address, msgs [][]byte, sigs []tpcrtypes.Signature) {
	var c CryptServiceEd25519
	secs := make([]tpcrtypes.PrivateKey, groupNum)
	pubs := make([]tpcrtypes.PublicKey, groupNum)
	addrs = make([]tpcrtypes.Address, groupNum)
	msgs = make([][]byte, groupNum)
	sigs = make([]tpcrtypes.Signature, groupNum)

	for i := 0; i < groupNum; i++ {
		secs[i], pubs[i], _ = c.GeneratePriPubKey()
	}
	for i := range pubs {
		addrs[i], _ = c.CreateAddress(pubs[i])
	}

	msgLen := 256
	for i := range msgs {
		msgs[i] = make([]byte, msgLen)
		_, _ = rand.Read(msgs[i])
	}
	for i := range msgs {
		sigs[i], _ = c.Sign(secs[i], msgs[i])
	}
	return addrs, msgs, sigs
}

func BenchmarkBatchVerify10(b *testing.B) {
	b.StopTimer()
	var c CryptServiceEd25519
	addrs, msgs, sigs := prepareForBatchVerify(benchGroupAmount10)
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		retBool, err := c.BatchVerify(addrs, msgs, sigs)
		b.StopTimer()
		assert.Equal(b, true, retBool, "BatchVerify err")
		assert.Equal(b, nil, err, "BatchVerify err")
	}
}

func BenchmarkBatchVerify50(b *testing.B) {
	b.StopTimer()
	var c CryptServiceEd25519
	addrs, msgs, sigs := prepareForBatchVerify(benchGroupAmount50)
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		retBool, err := c.BatchVerify(addrs, msgs, sigs)
		b.StopTimer()
		assert.Equal(b, true, retBool, "BatchVerify err")
		assert.Equal(b, nil, err, "BatchVerify err")
	}
}

func BenchmarkBatchVerify100(b *testing.B) {
	b.StopTimer()
	var c CryptServiceEd25519
	addrs, msgs, sigs := prepareForBatchVerify(benchGroupAmount100)
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		retBool, err := c.BatchVerify(addrs, msgs, sigs)
		b.StopTimer()
		assert.Equal(b, true, retBool, "BatchVerify err")
		assert.Equal(b, nil, err, "BatchVerify err")
	}
}
