package keyring

import (
	"fmt"
	"github.com/99designs/keyring"
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

func TestKeyring_File(t *testing.T) {
	var kri KeyringImp
	initArg := InitArg{
		EncryptWay: getTestEncrytWayInstance_ed25519(t),
		RootPath:   dirPathForTest(),
		Backend:    string(keyring.FileBackend),
		Cs:         crypt.CreateCryptService(nil, tpcrtypes.CryptType_Ed25519),
	}
	err := kri.Init(initArg)
	assert.Equal(t, nil, err, "init with backend:", keyring.FileBackend, "err:", err)

	testAddr := "whatever_Addr"
	item := key_store.KeyItem{
		Seckey:    []byte("whatever"),
		CryptType: tpcrtypes.CryptType_Ed25519, // Any valid type is fine.
	}

	err = kri.SetEnable(true)
	assert.Nil(t, err, "SetWalletEnable err:", err)
	err = kri.SetAddr(testAddr, item)
	assert.Equal(t, nil, err, "set addr err:", err)
	getItem, err := kri.GetAddr(testAddr)
	assert.Equal(t, nil, err, "get addr err:", err)
	assert.Equal(t, item.CryptType, getItem.CryptType)
	for i := range item.Seckey {
		assert.Equal(t, item.Seckey[i], getItem.Seckey[i])
	}

	_, err = kri.GetEnable()
	assert.Equal(t, nil, err, "get wallet enable err:", err)

	err = kri.Remove(testAddr)
	assert.Nil(t, err, "remove addr err:", err)
	err = kri.Remove(key_store.EnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)

	err = os.RemoveAll(filepath.Join(dirPathForTest(), keysFolderName))
	assert.Nil(t, err, "remove test key folder err")
}

func testSetGetRemove(kri *KeyringImp, t *testing.T) {
	testAddr := "whatever_Addr"
	item := key_store.KeyItem{
		Seckey:    []byte("whatever"),
		CryptType: tpcrtypes.CryptType_Ed25519, // Any valid CryptType is fine.
	}

	err := kri.SetEnable(true)
	assert.Nil(t, err, "SetWalletEnable err:", err)
	err = kri.SetAddr(testAddr, item)
	assert.Equal(t, nil, err, "set addr err:", err)
	getItem, err := kri.GetAddr(testAddr)
	assert.Equal(t, nil, err, "get addr err:", err)
	assert.Equal(t, item.CryptType, getItem.CryptType)
	for i := range item.Seckey {
		assert.Equal(t, item.Seckey[i], getItem.Seckey[i])
	}

	_, err = kri.GetEnable()
	assert.Equal(t, nil, err, "get wallet enable err:", err)

	err = kri.Remove(testAddr)
	assert.Nil(t, err, "remove addr err:", err)
	err = kri.Remove(key_store.EnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)

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
