package wallet

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestTopiaFileKeyStoreFunction(t *testing.T) {
	cleanCache()

	var fks fileKeyStore

	testAddr := "whatever_Addr"
	initArgument := initArg{
		RootPath:           testFolderPath(),
		EncryptWayOfWallet: getTestEncrytWayInstance_secp256(t),
	}
	defaultAddr := "whatever_defaultAddr"

	err := fks.Init(initArgument)
	assert.Nil(t, err, "fileKeyStore init err", err)

	item := keyItem{
		Seckey:    []byte("whatever"),
		CryptType: tpcrtypes.CryptType_Ed25519, // Any valid type is fine.
	}
	err = fks.SetEnable(true)
	assert.Nil(t, err, "fileKeyStore SetWalletEnable err", err)

	err = setDefaultAddr(defaultAddr)
	assert.Nil(t, err, "fileKeyStore SetDefaultAddr err:", err)
	err = fks.SetAddr(testAddr, item)
	assert.Equal(t, nil, err, "fileKeyStore set addr err:", err)
	getItem, err := fks.GetAddr(testAddr)
	assert.Equal(t, nil, err, "fileKeyStore get addr err:", err)
	assert.Equal(t, item.CryptType, getItem.CryptType)
	for i := range item.Seckey {
		assert.Equal(t, item.Seckey[i], getItem.Seckey[i])
	}

	_, err = fks.GetEnable()
	assert.Equal(t, nil, err, "fileKeyStore get wallet enable err:", err)
	getDA := getDefaultAddr()
	assert.Equal(t, defaultAddr, getDA, "fileKeyStore get default addr err:", err)

	err = fks.Remove(testAddr)
	assert.Nil(t, err, "remove addr err:", err)
	err = fks.Remove(walletEnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)

	err = removeDefaultAddr()
	assert.Nil(t, err, "remove defaultAddrKey err:", err)

	err = os.RemoveAll(filepath.Join(testFolderPath(), topiaKeysFolderName))
	assert.Nil(t, err, "remove test key folder err")
}
