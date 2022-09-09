package file_key_store

import (
	"bytes"
	"fmt"
	"github.com/TopiaNetwork/topia/crypt"
	"github.com/TopiaNetwork/topia/crypt/ed25519"
	"github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/wallet/key_store"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileKeyStoreFunction(t *testing.T) {
	var fks FileKeyStore

	testAddr := "whatever_Addr"
	initArgument := InitArg{
		RootPath:   dirPathForTest(),
		EncryptWay: getTestEncrytWayInstance_secp256(t),
		Cs:         crypt.CreateCryptService(nil, tpcrtypes.CryptType_Secp256),
	}

	err := fks.Init(initArgument)
	assert.Nil(t, err, "fileKeyStore init err", err)

	item := key_store.KeyItem{
		Seckey:    []byte("whatever"),
		CryptType: tpcrtypes.CryptType_Ed25519, // Any valid type is fine.
	}
	err = fks.SetEnable(true)
	assert.Nil(t, err, "fileKeyStore SetWalletEnable err", err)

	err = fks.SetDefaultAddr(testAddr)
	assert.Nil(t, err)
	getDefaultAddr, err := fks.GetDefaultAddr()
	assert.Nil(t, err)
	assert.Equal(t, testAddr, getDefaultAddr)

	err = fks.SetAddr(testAddr, item)
	assert.Equal(t, nil, err, "fileKeyStore set addr err:", err)
	getItem, err := fks.GetAddr(testAddr)
	assert.Equal(t, nil, err, "fileKeyStore get addr err:", err)
	assert.Equal(t, item.CryptType, getItem.CryptType)

	equal := bytes.Equal(item.Seckey, getItem.Seckey)
	assert.Equal(t, true, equal)

	_, err = fks.GetEnable()
	assert.Equal(t, nil, err, "fileKeyStore get wallet enable err:", err)

	err = fks.Remove(testAddr)
	assert.Nil(t, err, "remove addr err:", err)
	err = fks.Remove(key_store.EnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)
	err = fks.Remove(key_store.DefaultAddrKey)
	assert.Nil(t, err)

	keyStorePath := filepath.Join(dirPathForTest(), keysFolderName)
	err = key_store.SetDirPerm(filepath.Dir(keyStorePath), filepath.Base(keyStorePath))
	assert.Nil(t, err)
	err = os.RemoveAll(keyStorePath)
	assert.Nil(t, err, "remove test key folder err")
}

func getTestEncrytWayInstance_ed25519(t *testing.T) EncryptWay {
	var c ed25519.CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Nil(t, err, err)
	return EncryptWay{
		CryptType: tpcrtypes.CryptType_Ed25519,
		Pubkey:    pub,
		Seckey:    sec,
	}
}

func getTestEncrytWayInstance_secp256(t *testing.T) EncryptWay {
	var c secp256.CryptServiceSecp256
	sec, pub, err := c.GeneratePriPubKey()
	assert.Nil(t, err, err)
	return EncryptWay{
		CryptType: tpcrtypes.CryptType_Secp256,
		Pubkey:    pub,
		Seckey:    sec,
	}
}

// DirPathForTest return path string of the folder which this file is in.
func dirPathForTest() string {
	var testFolderPath string
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		testFolderPath = path.Dir(filename)
	} else {
		fmt.Println("get file path err")
		return ""
	}
	return testFolderPath
}
