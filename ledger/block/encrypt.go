package block

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"io"
	"os"
)



func (df *TopiaFile)AesEncrypt(key string) string {

	filedata, _ := os.OpenFile(df.File.Name(), os.O_RDWR, 0644)
	origData := bytes.NewBuffer(nil)
	if _, err := io.Copy(origData, filedata); err != nil {
		return ""
	}
	k := []byte(key)


	block, _ := aes.NewCipher(k)

	blockSize := block.BlockSize()

	origDatabyte := PKCS7Padding(origData.Bytes(), blockSize)

	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])

	cryted := make([]byte, len(origDatabyte))

	blockMode.CryptBlocks(cryted, origDatabyte)

	return base64.StdEncoding.EncodeToString(cryted)

}

func (df *TopiaFile)AesDecrypt(cryted string, key string) string {

	crytedByte, _ := base64.StdEncoding.DecodeString(cryted)
	k := []byte(key)


	block, _ := aes.NewCipher(k)

	blockSize := block.BlockSize()

	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])

	orig := make([]byte, len(crytedByte))

	blockMode.CryptBlocks(orig, crytedByte)

	orig = PKCS7UnPadding(orig)
	return string(orig)
}


func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}


func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
