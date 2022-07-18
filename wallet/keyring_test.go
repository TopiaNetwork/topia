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
	var kri keyringImp
	err := initWithBackendX(&kri, keyring.FileBackend, testFolderPath())
	assert.Equal(t, nil, err, "init with backend:", keyring.FileBackend, "err:", err)
	item := addrItem{
		Addr:       "whatever",
		AddrLocked: false,
		Seckey:     []byte("whatever"),
		Pubkey:     []byte("whatever"),
		CryptType:  tpcrtypes.CryptType_Ed25519, // Any valid type is fine.
	}

	err = kri.SetWalletEnable(true)
	assert.Nil(t, err, "SetWalletEnable err:", err)
	err = kri.SetDefaultAddr(item.Addr)
	assert.Nil(t, err, "SetDefaultAddr err:", err)
	err = kri.SetAddr(item)
	assert.Equal(t, nil, err, "set addr err:", err)
	getItem, err := kri.GetAddr(item.Addr)
	assert.Equal(t, nil, err, "get addr err:", err)
	assert.Equal(t, item.AddrLocked, getItem.AddrLocked)
	assert.Equal(t, item.CryptType, getItem.CryptType)
	for i := range item.Seckey {
		assert.Equal(t, item.Seckey[i], getItem.Seckey[i])
	}
	for i := range item.Pubkey {
		assert.Equal(t, item.Pubkey[i], getItem.Pubkey[i])
	}

	_, err = kri.GetWalletEnable()
	assert.Equal(t, nil, err, "get wallet enable err:", err)
	_, err = kri.GetDefaultAddr()
	assert.Equal(t, nil, err, "get default addr err:", err)

	err = kri.RemoveItem(item.Addr)
	assert.Nil(t, err, "remove addr err:", err)
	err = kri.RemoveItem(walletEnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)
	err = kri.RemoveItem(defaultAddrKey)
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
	kri.mutex.Lock()
	defer kri.mutex.Unlock()

	tempKeyring, err := keyring.Open(config)
	if err != nil {
		return errors.New("open keyring err: " + err.Error())
	}
	kri.k = tempKeyring

	if _, err = kri.k.Get(walletEnableKey); err != nil { // if walletEnableKey hasn't been set, set it
		tempItem := keyring.Item{
			Key:         walletEnableKey,
			Data:        []byte(walletEnabled),
			Label:       "",
			Description: "",
		}
		if err = kri.k.Set(tempItem); err != nil {
			return errors.New("set walletEnable Item err: " + err.Error())
		}
	}

	if _, err = kri.k.Get(defaultAddrKey); err != nil { // if walletDefaultAddrKey hasn't been set, set it
		tempItem := keyring.Item{
			Key:         defaultAddrKey,
			Data:        []byte("please_set_default_address"),
			Label:       "",
			Description: "",
		}
		if err = kri.k.Set(tempItem); err != nil {
			return errors.New("set walletDefaultAddr Item err: " + err.Error())
		}
	}

	return nil
}

func testSetGetRemove(kri keyringImp, t *testing.T) {
	item := addrItem{
		Addr:       "whatever",
		AddrLocked: false,
		Seckey:     []byte("whatever"),
		Pubkey:     []byte("whatever"),
		CryptType:  tpcrtypes.CryptType_Ed25519, // Any valid CryptType is fine.
	}

	err := kri.SetWalletEnable(true)
	assert.Nil(t, err, "SetWalletEnable err:", err)
	err = kri.SetDefaultAddr(item.Addr)
	assert.Nil(t, err, "SetDefaultAddr err:", err)
	err = kri.SetAddr(item)
	assert.Equal(t, nil, err, "set addr err:", err)
	getItem, err := kri.GetAddr(item.Addr)
	assert.Equal(t, nil, err, "get addr err:", err)
	assert.Equal(t, item.AddrLocked, getItem.AddrLocked)
	assert.Equal(t, item.CryptType, getItem.CryptType)
	for i := range item.Seckey {
		assert.Equal(t, item.Seckey[i], getItem.Seckey[i])
	}
	for i := range item.Pubkey {
		assert.Equal(t, item.Pubkey[i], getItem.Pubkey[i])
	}

	_, err = kri.GetWalletEnable()
	assert.Equal(t, nil, err, "get wallet enable err:", err)
	_, err = kri.GetDefaultAddr()
	assert.Equal(t, nil, err, "get default addr err:", err)

	err = kri.RemoveItem(item.Addr)
	assert.Nil(t, err, "remove addr err:", err)
	err = kri.RemoveItem(walletEnableKey)
	assert.Nil(t, err, "remove walletEnableKey err:", err)
	err = kri.RemoveItem(defaultAddrKey)
	assert.Nil(t, err, "remove defaultAddrKey err:", err)

}
