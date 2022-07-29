package wallet

import (
	"errors"
	"github.com/99designs/keyring"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyring_File(t *testing.T) {
	cleanCache()

	var kri keyringImp
	err := initWithBackendX(&kri, keyring.FileBackend, testFolderPath())
	assert.Equal(t, nil, err, "init with backend:", keyring.FileBackend, "err:", err)

	testAddr := "whatever_Addr"
	item := keyItem{
		Seckey:    []byte("whatever"),
		CryptType: tpcrtypes.CryptType_Ed25519, // Any valid type is fine.
	}
	defaultAddr := "whatever_defaultAddr"

	err = kri.SetEnable(true)
	assert.Nil(t, err, "SetWalletEnable err:", err)
	err = setDefaultAddr(defaultAddr)
	assert.Nil(t, err, "SetDefaultAddr err:", err)
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
	getDA := getDefaultAddr()
	assert.Equal(t, defaultAddr, getDA, "get default addr err:", err)

	err = kri.Remove(testAddr)
	assert.Nil(t, err, "remove addr err:", err)
	err = kri.Remove(walletEnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)
	err = removeDefaultAddr()
	assert.Nil(t, err, "remove defaultAddrKey err:", err)

	err = os.RemoveAll(filepath.Join(testFolderPath(), keyringKeysFolderName))
	assert.Nil(t, err, "remove test key folder err")
}

func initWithBackendX(kri *keyringImp, bkd keyring.BackendType, fileFolderPath string) error {
	if isValidFolderPath(fileFolderPath) == false {
		return errors.New("input fileFolderPath is not a valid folder path")
	}
	config := keyring.Config{
		AllowedBackends:                []keyring.BackendType{bkd},
		ServiceName:                    "TopiaWalletStore",
		KeychainName:                   filepath.Join(testFolderPath(), keyringKeysFolderName, "TopiaWallet"),
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         true,
		KeychainAccessibleWhenUnlocked: true,
		KeychainPasswordFunc:           keyring.FixedStringPrompt("password"),
		FilePasswordFunc:               keyring.FixedStringPrompt("password"),
		FileDir:                        filepath.Join(fileFolderPath, keyringKeysFolderName),
		KeyCtlScope:                    "thread",
		KeyCtlPerm:                     0x3f3f0000, // "alswrvalswrv------------",
		KWalletAppID:                   "TopiaKWalletApp",
		KWalletFolder:                  "TopiaKWallet",
		WinCredPrefix:                  "", // default "keyring"
	}

	tempKeyring, err := keyring.Open(config)
	if err != nil {
		return errors.New("open keyring err: " + err.Error())
	}
	kri.k = tempKeyring

	if _, err = kri.getEnableFromBackend(); err != nil { // if walletEnableKey hasn't been set, set it
		err = kri.SetEnable(true)
		if err != nil {
			return err
		}
	}

	return nil
}

func testSetGetRemove(kri keyringImp, t *testing.T) {
	testAddr := "whatever_Addr"
	item := keyItem{
		Seckey:    []byte("whatever"),
		CryptType: tpcrtypes.CryptType_Ed25519, // Any valid CryptType is fine.
	}

	defaultAddr := "whatever_defaultAddr"

	err := kri.SetEnable(true)
	assert.Nil(t, err, "SetWalletEnable err:", err)
	err = setDefaultAddr(defaultAddr)
	assert.Nil(t, err, "SetDefaultAddr err:", err)
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
	getDA := getDefaultAddr()
	assert.Equal(t, defaultAddr, getDA, "get default addr err:", err)

	err = kri.Remove(testAddr)
	assert.Nil(t, err, "remove addr err:", err)
	err = kri.Remove(walletEnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)

	err = removeDefaultAddr()
	assert.Nil(t, err, "remove defaultAddrKey err:", err)

}
