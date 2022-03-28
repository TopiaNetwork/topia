package bls12381

import (
	"bytes"
	"testing"
)

func TestGeneratePriPubKey(t *testing.T) {
	var c CryptServiceBLS12381
	_, _, err := c.GeneratePriPubKey()
	if err != nil {
		t.Error("TestGeneratePriPubKey failed: ", err)
	}
}

func TestConvertToPublic(t *testing.T) {
	var c CryptServiceBLS12381
	sec, pub, err := c.GeneratePriPubKey()
	if err != nil {
		t.Error("TestConvertToPublic GeneratePriPubKey failed: ", err)
	}
	pubConvert, err := c.ConvertToPublic(sec)
	if err != nil {
		t.Error("TestConvertToPublic ConvertToPublic failed: ", err)
	}
	if bytes.Equal(pub, pubConvert) != true {
		t.Error("TestConvertToPublic ConvertToPublic failed.")
	}
}

func TestSign(t *testing.T) {
	var c CryptServiceBLS12381
	sec, _, err := c.GeneratePriPubKey()
	if err != nil {
		t.Error("TestSign GeneratePriPubKey failed: ", err)
	}
	msgToSign := "This is the msg for sign."
	_, err = c.Sign(sec, []byte(msgToSign))
	if err != nil {
		t.Error("TestSign Sign failed: ", err)
	}
}

func TestVerify(t *testing.T) {
	var c CryptServiceBLS12381
	sec, pub, err := c.GeneratePriPubKey()
	if err != nil {
		t.Error("TestVerify GeneratePriPubKey failed: ", err)
	}
	msgToSign := "This is the msg for sign."
	sig, err := c.Sign(sec, []byte(msgToSign))
	if err != nil {
		t.Error("TestVerify Sign failed: ", err)
	}
	if b, err := c.Verify(pub, []byte(msgToSign), sig); b != true || err != nil {
		t.Error("TestVerify Verify failed: ", err)
	}
}

func TestCreateAddress(t *testing.T) {
	var c CryptServiceBLS12381
	_, pub, err := c.GeneratePriPubKey()
	if err != nil {
		t.Error("TestCreateAddress GeneratePriPubKey failed: ", err)
	}
	if _, err := c.CreateAddress(pub); err != nil {
		t.Error("TestCreateAddress CreateAddress failed: ", err)
	}
}
