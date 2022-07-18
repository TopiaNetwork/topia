package bls12381

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGeneratePriPubKey(t *testing.T) {
	var c CryptServiceBLS12381
	_, _, err := c.GeneratePriPubKey()
	assert.Equal(t, nil, err, "TestGeneratePriPubKey failed.")
}

func TestConvertToPublic(t *testing.T) {
	var c CryptServiceBLS12381
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, nil, err, "TestConvertToPublic GeneratePriPubKey failed.")

	pubConvert, err := c.ConvertToPublic(sec)
	assert.Equal(t, nil, err, "TestConvertToPublic ConvertToPublic failed.")
	assert.Equal(t, true, bytes.Equal(pub, pubConvert), "TestConvertToPublic ConvertToPublic failed.")
}

func TestSign(t *testing.T) {
	var c CryptServiceBLS12381
	sec, _, err := c.GeneratePriPubKey()
	assert.Equal(t, nil, err, "TestSign GeneratePriPubKey failed.")

	msgToSign := "This is the msg for sign."
	_, err = c.Sign(sec, []byte(msgToSign))
	assert.Equal(t, nil, err, "TestSign Sign failed.")
}

func TestVerify(t *testing.T) {
	var c CryptServiceBLS12381
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, nil, err, "TestVerify GeneratePriPubKey failed.")

	msgToSign := "This is the msg for sign."
	sig, err := c.Sign(sec, []byte(msgToSign))
	assert.Equal(t, nil, err, "TestVerify Sign failed.")

	addr, err := c.CreateAddress(pub)
	assert.Nil(t, err, "TestVerify CreateAddress failed.")

	b, err := c.Verify(addr, []byte(msgToSign), sig)
	assert.Equal(t, true, b, "TestVerify Verify failed.")
	assert.Equal(t, nil, err, "TestVerify Verify failed.")
}

func TestCreateAddress(t *testing.T) {
	var c CryptServiceBLS12381
	_, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, nil, err, "TestCreateAddress GeneratePriPubKey failed.")

	_, err = c.CreateAddress(pub)
	assert.Equal(t, nil, err, "TestCreateAddress CreateAddress failed.")
}

func TestStreamEncryptDecrypt(t *testing.T) {
	var c CryptServiceBLS12381
	msg := "This is the message to be en-and-de-crypted"

	sec, pub, err := c.GeneratePriPubKey()
	assert.Nil(t, err, "GeneratePriPubKey err", err)

	encryptedData, err := c.StreamEncrypt(pub, []byte(msg))
	assert.Nil(t, err, "StreamEncrypt err", err)

	decryptedMsg, err := c.StreamDecrypt(sec, encryptedData)
	assert.Nil(t, err, "StreamDecrypt err", err)
	assert.Equal(t, msg, string(decryptedMsg), "decryptedMsg is not equal to msg")

	secWrong, _, _ := c.GeneratePriPubKey()
	wrongDe, err := c.StreamDecrypt(secWrong, encryptedData)
	assert.NotNil(t, err, "StreamDecrypt err", err)
	assert.NotEqual(t, msg, string(wrongDe), "wrong key shouldn't decrypt right")
}
