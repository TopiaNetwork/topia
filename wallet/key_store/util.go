package key_store

import (
	"errors"
	"github.com/TopiaNetwork/topia/crypt"
	"github.com/TopiaNetwork/topia/crypt/ed25519"
	"github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"os"
)

func IsValidFolderPath(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	if s.IsDir() == false {
		return false
	}
	return true
}

func IsPathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func IsValidTopiaAddr(addr tpcrtypes.Address) bool {
	_, err := addr.CryptType()
	if err != nil {
		return false
	}
	return true
}

func SetDirReadOnly(path string, dirName string) error {
	userPath, _ := os.Getwd()
	err := os.Chdir(path)
	if err != nil {
		return err
	}
	err = os.Chmod(dirName, 0544)
	if err != nil {
		return err
	}
	err = os.Chdir(userPath)
	if err != nil {
		return err
	}
	return nil
}

func SetDirPerm(path string, dirName string) error {
	userPath, _ := os.Getwd()
	err := os.Chdir(path)
	if err != nil {
		return err
	}
	err = os.Chmod(dirName, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.Chdir(userPath)
	if err != nil {
		return err
	}
	return nil
}

func StreamEncrypt(cryptType tpcrtypes.CryptType, pubKey tpcrtypes.PublicKey, data []byte) (encryptedData []byte, err error) {
	var c crypt.CryptService
	switch cryptType {
	case tpcrtypes.CryptType_Secp256:
		c = new(secp256.CryptServiceSecp256)
	case tpcrtypes.CryptType_Ed25519:
		c = new(ed25519.CryptServiceEd25519)
	default:
		return nil, errors.New("unsupported CryptType")
	}
	return c.StreamEncrypt(pubKey, data)
}

func StreamDecrypt(cryptType tpcrtypes.CryptType, secKey tpcrtypes.PrivateKey, data []byte) (decryptedData []byte, err error) {
	var c crypt.CryptService
	switch cryptType {
	case tpcrtypes.CryptType_Secp256:
		c = new(secp256.CryptServiceSecp256)
	case tpcrtypes.CryptType_Ed25519:
		c = new(ed25519.CryptServiceEd25519)
	default:
		return nil, errors.New("unsupported CryptType")
	}
	return c.StreamDecrypt(secKey, data)
}
