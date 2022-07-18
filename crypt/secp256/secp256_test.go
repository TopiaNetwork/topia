package secp256

import (
	"crypto/rand"
	"crypto/sha256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGeneratePriPubKey(t *testing.T) {
	var c CryptServiceSecp256
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "Generate seckey length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "Generate pubkey length err")
	assert.Equal(t, nil, err, "Generate key err")
}

func TestGeneratePriPubKeyBySeed(t *testing.T) {
	var c CryptServiceSecp256
	seed := make([]byte, SeedBytes)
	_, err := rand.Read(seed)
	assert.Equal(t, nil, err, "generate rand num err")
	sec, pub, err := c.GeneratePriPubKeyBySeed(seed)
	assert.Equal(t, PrivateKeyBytes, len(sec), "Generate seckey length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "Generate pubkey length err")
	assert.Equal(t, nil, err, "Generate key err")
}

func TestConvertToPublic(t *testing.T) {
	var c CryptServiceSecp256
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "Generate seckey length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "Generate pubkey length err")
	assert.Equal(t, nil, err, "Generate key err")

	pubConvert, err := c.ConvertToPublic(sec)
	for i := range pub {
		assert.Equal(t, pub[i], pubConvert[i], "Convert to pubkey err")
	}
}

func TestSignAndVerify(t *testing.T) {
	var c CryptServiceSecp256
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "Generate seckey length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "Generate pubkey length err")
	assert.Equal(t, nil, err, "Generate key err")

	msgArr := sha256.Sum256([]byte("This is the message for sign"))
	sig, err := c.Sign(sec, msgArr[:])
	assert.Equal(t, nil, err, "Sign err")

	addr, err := c.CreateAddress(pub)
	assert.Nil(t, err, "CreateAddress err")

	retBool, err := c.Verify(addr, msgArr[:], sig)
	assert.Equal(t, true, retBool, "Verify should be true")
	assert.Equal(t, nil, err, "Verify err")
}

func TestRecoverPublicKey(t *testing.T) {
	var c CryptServiceSecp256
	sec, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, PrivateKeyBytes, len(sec), "Generate seckey length err")
	assert.Equal(t, PublicKeyBytes, len(pub), "Generate pubkey length err")
	assert.Equal(t, nil, err, "Generate key err")

	msgArr := sha256.Sum256([]byte("This is the message for sign"))
	sig, err := c.Sign(sec, msgArr[:])
	assert.Equal(t, nil, err, "Sign err")

	recPubkey, err := c.RecoverPublicKey(msgArr[:], sig)
	assert.Equal(t, nil, err, "Recover PublicKey err")
	for i := range pub {
		assert.Equal(t, pub[i], recPubkey[i], "Recover PublicKey err")
	}
}

func TestCreateAddress(t *testing.T) {
	var c CryptServiceSecp256
	_, pub, err := c.GeneratePriPubKey()
	assert.Equal(t, nil, err, "TestCreateAddress GeneratePriPubKey failed")

	_, err = c.CreateAddress(pub)
	assert.Equal(t, nil, err, "TestCreateAddress CreateAddress failed")
}

func TestCompatibleWithETH(t *testing.T) {
	secETH := [][]byte{
		{246, 89, 111, 83, 168, 133, 90, 202, 222, 83, 24, 50, 224, 88, 17, 213, 22, 186, 53, 120, 84, 65, 144, 71, 156, 6, 225, 175, 156, 176, 19, 169},
		{46, 152, 44, 121, 4, 41, 243, 182, 210, 147, 195, 232, 27, 159, 165, 143, 115, 203, 120, 217, 251, 214, 131, 235, 227, 161, 220, 244, 211, 130, 188, 111},
		{214, 195, 18, 243, 74, 186, 15, 4, 144, 232, 70, 184, 223, 181, 207, 225, 114, 157, 73, 82, 226, 233, 232, 32, 165, 239, 17, 251, 171, 200, 30, 221},
	}
	pubETH := [][]byte{
		{4, 225, 111, 182, 98, 98, 83, 90, 17, 3, 178, 189, 155, 65, 57, 226, 103, 83, 159, 39, 105, 18, 132, 180, 118, 41, 236, 90, 201, 188, 116, 74, 213, 208, 189, 172, 13, 190, 170, 105, 70, 208, 133, 203, 62, 2, 115, 190, 14, 164, 189, 183, 29, 175, 172, 145, 81, 188, 111, 192, 122, 159, 91, 183, 249},
		{4, 92, 18, 57, 22, 208, 212, 26, 180, 142, 16, 90, 151, 112, 51, 48, 120, 237, 26, 255, 154, 241, 6, 235, 34, 172, 164, 224, 35, 21, 15, 196, 22, 27, 255, 252, 170, 229, 69, 117, 232, 177, 145, 189, 213, 37, 153, 93, 179, 145, 67, 93, 47, 101, 201, 232, 172, 220, 8, 62, 167, 128, 57, 168, 70},
		{4, 56, 163, 192, 226, 1, 99, 243, 131, 61, 6, 174, 203, 165, 66, 17, 176, 167, 28, 227, 175, 237, 66, 9, 50, 238, 99, 128, 229, 12, 107, 78, 14, 236, 222, 207, 224, 215, 198, 129, 4, 229, 180, 35, 173, 23, 49, 49, 70, 210, 108, 53, 158, 106, 166, 248, 123, 146, 115, 234, 71, 54, 68, 97, 220},
	}
	sigETH := [][]byte{
		{68, 203, 235, 138, 220, 185, 114, 92, 88, 161, 165, 54, 71, 96, 57, 24, 106, 213, 134, 120, 193, 120, 121, 223, 1, 57, 64, 89, 120, 52, 181, 79, 75, 226, 234, 137, 29, 225, 57, 142, 51, 107, 235, 40, 57, 59, 254, 185, 217, 181, 119, 86, 171, 113, 124, 212, 71, 92, 205, 163, 240, 217, 195, 62, 1},
		{162, 41, 116, 21, 6, 48, 168, 23, 232, 47, 122, 107, 157, 200, 102, 194, 96, 99, 30, 7, 217, 105, 62, 62, 145, 234, 28, 19, 210, 195, 173, 255, 24, 131, 7, 91, 219, 73, 102, 202, 173, 232, 61, 33, 50, 211, 64, 7, 38, 15, 69, 238, 50, 208, 188, 246, 203, 95, 117, 171, 98, 201, 138, 177, 0},
		{113, 120, 114, 201, 26, 98, 89, 68, 186, 199, 181, 225, 79, 237, 26, 67, 192, 204, 106, 5, 77, 245, 122, 7, 23, 14, 196, 239, 142, 106, 21, 211, 54, 111, 17, 225, 206, 126, 131, 225, 38, 169, 235, 170, 94, 138, 183, 48, 167, 245, 157, 47, 110, 75, 66, 129, 46, 207, 116, 54, 227, 16, 215, 173, 0},
	}
	msg := [][]byte{
		{89, 228, 239, 96, 91, 54, 208, 175, 5, 153, 79, 130, 28, 140, 244, 155, 59, 168, 81, 63, 87, 87, 57, 14, 133, 221, 57, 211, 156, 69, 27, 0},
		{84, 25, 98, 2, 194, 227, 248, 77, 49, 11, 162, 190, 148, 149, 158, 203, 229, 129, 104, 196, 250, 53, 138, 33, 113, 11, 130, 52, 197, 39, 159, 70},
		{242, 182, 115, 165, 20, 14, 31, 149, 224, 137, 68, 105, 203, 148, 42, 116, 198, 215, 189, 183, 190, 25, 240, 229, 215, 73, 251, 212, 203, 55, 112, 129},
	}
	var c CryptServiceSecp256
	for i := range secETH {
		pubConvert, err := c.ConvertToPublic(secETH[i])
		assert.Equal(t, nil, err, "ETH seckey convert to public key err")
		for j := range pubETH[i] {
			assert.Equal(t, pubETH[i][j], pubConvert[j], "ETH convert to pubkey err")
		}
	}

	addrs := make([]tpcrtypes.Address, len(secETH))

	for i := range secETH {
		sign, err := c.Sign(secETH[i], msg[i])
		assert.Equal(t, nil, err, "ETH sign err")
		for j := range sigETH[i] {
			assert.Equal(t, sigETH[i][j], sign[j], "ETH sign err")
		}

		tempAddr, err := c.CreateAddress(pubETH[i])
		assert.Nil(t, err, "CreateAddress err")
		addrs[i] = tempAddr

		retBool, err := c.Verify(addrs[i], msg[i], sign)
		assert.Equal(t, nil, err, "ETH verify err")
		assert.Equal(t, true, retBool, "ETH verify false")

		pubRecover, err := c.RecoverPublicKey(msg[i], sigETH[i])
		assert.Equal(t, nil, err, "ETH recover public key err")
		for j := range pubETH[i] {
			assert.Equal(t, pubETH[i][j], pubRecover[j], "ETH recover public key err")
		}
	}
}

func TestStreamEncryptDecrypt(t *testing.T) {
	var c CryptServiceSecp256
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
