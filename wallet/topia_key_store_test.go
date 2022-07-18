package wallet

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestTopiaFileKeyStoreFunction(t *testing.T) {
	var fks fileKeyStore

	initArgument := initArg{
		RootPath:           testFolderPath(),
		EncryptWayOfWallet: getTestEncrytWayInstance_ed25519(t),
	}
	err := fks.Init(initArgument)
	assert.Nil(t, err, "fileKeyStore init err", err)

	item := addrItem{
		Addr:       "whatever",
		AddrLocked: false,
		Seckey:     []byte("whatever"),
		Pubkey:     []byte("whatever"),
		CryptType:  tpcrtypes.CryptType_Ed25519, // Any valid type is fine.
	}
	err = fks.SetWalletEnable(true)
	assert.Nil(t, err, "fileKeyStore SetWalletEnable err", err)

	err = fks.SetDefaultAddr(item.Addr)
	assert.Nil(t, err, "fileKeyStore SetDefaultAddr err:", err)
	err = fks.SetAddr(item)
	assert.Equal(t, nil, err, "fileKeyStore set addr err:", err)
	getItem, err := fks.GetAddr(item.Addr)
	assert.Equal(t, nil, err, "fileKeyStore get addr err:", err)
	assert.Equal(t, item.AddrLocked, getItem.AddrLocked)
	assert.Equal(t, item.CryptType, getItem.CryptType)
	for i := range item.Seckey {
		assert.Equal(t, item.Seckey[i], getItem.Seckey[i])
	}
	for i := range item.Pubkey {
		assert.Equal(t, item.Pubkey[i], getItem.Pubkey[i])
	}

	_, err = fks.GetWalletEnable()
	assert.Equal(t, nil, err, "fileKeyStore get wallet enable err:", err)
	_, err = fks.GetDefaultAddr()
	assert.Equal(t, nil, err, "fileKeyStore get default addr err:", err)

	err = fks.RemoveItem(item.Addr)
	assert.Nil(t, err, "remove addr err:", err)
	err = fks.RemoveItem(walletEnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)
	err = fks.RemoveItem(defaultAddrKey)
	assert.Nil(t, err, "remove defaultAddrKey err:", err)

	err = os.RemoveAll(filepath.Join(testFolderPath(), topiaKeysFolderName))
	assert.Nil(t, err, "remove test key folder err")
}
