package wallet

import (
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/crypt/ed25519"
	"github.com/TopiaNetwork/topia/crypt/secp256"
	"github.com/TopiaNetwork/topia/crypt/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWallet_Function(t *testing.T) {

	ews := []EncryptWayOfWallet{getTestEncrytWayInstance_ed25519(t), getTestEncrytWayInstance_secp256(t)}
	for i := range ews {
		cleanCache()

		var keyFolderName string

		walletBackendConfig.RootPath = testFolderPath() // only for test
		w, err := NewWallet(tplogcmm.NoLevel, nil, ews[i])
		assert.Nil(t, err, "NewWallet err")

		wImp, ok := w.(*wallet)
		assert.Equal(t, true, ok, "cannot happen")

		switch wImp.ks.(type) {
		case *fileKeyStore:
			keyFolderName = topiaKeysFolderName
		case *keyringImp:
			keyFolderName = keyringKeysFolderName
		default:
			t.Fatal("cannot happen")
		}

		addr, err := w.Create(types.CryptType_Ed25519)
		assert.Equal(t, nil, err, "create addr err:", err)

		_, err = w.Sign(addr, []byte("msg to sign"))
		assert.Equal(t, nil, err, "Sign err:", err)

		err = w.SetDefault(addr)
		assert.Equal(t, nil, err, "SetDefault addr err:", err)

		defaultAddr, err := w.Default()
		assert.Equal(t, addr, defaultAddr, "get Default addr err:", err)
		assert.Equal(t, nil, err, "get Default addr err:", err)

		_, err = w.Export(addr)
		assert.Equal(t, nil, err, "Export seckey err:", err)

		err = w.Lock(addr, true)
		assert.Equal(t, nil, err, "Lock addr err", err)

		bo, err := w.IsLocked(addr)
		assert.Equal(t, true, bo, "check addr lock state err", err)
		assert.Equal(t, nil, err, "check addr lock state err", err)

		err = w.Lock(addr, false)
		assert.Equal(t, nil, err, "Lock addr err", err)

		err = w.Delete(addr)
		assert.Equal(t, nil, err, "delete addr err", err)

		bo, err = w.Has(addr)
		assert.Equal(t, false, bo, "check Has err", err)

		_, err = w.List()
		assert.Equal(t, nil, err, "List err", err)

		err = w.Enable(false)
		assert.Equal(t, nil, err, "enable wallet err", err)

		bo, err = w.IsEnable()
		assert.Equal(t, false, bo, "check wallet enable state err", err)
		assert.Equal(t, nil, err, "check wallet enable state err", err)

		err = w.Enable(true)
		assert.Equal(t, nil, err, "enable wallet err", err)

		err = removeTestWalletFolder(w, filepath.Join(testFolderPath(), keyFolderName))
		assert.Nil(t, err, err)
	}

}

func TestMnemonic(t *testing.T) {
	cleanCache()

	passphrase := "this is test passphrase"

	var ct = tpcrtypes.CryptType_Ed25519

	walletBackendConfig.RootPath = testFolderPath() // only for test
	w, err := NewWallet(tplogcmm.NoLevel, nil, getTestEncrytWayInstance_ed25519(t))

	assert.Nil(t, err, "NewWallet err", err)

	mnemonic12, err := w.CreateMnemonic(ct, passphrase, 12)
	assert.Nil(t, err, "CreateMnemonic err", err)

	addr12, err := w.Recovery(ct, mnemonic12, passphrase)
	assert.Nil(t, err, "recover addr by mnemonic err", err)

	err = w.Delete(addr12)
	assert.Nil(t, err, "delete addr err", err)

	ct = tpcrtypes.CryptType_Secp256

	mnemonic24, err := w.CreateMnemonic(ct, passphrase, 24)
	assert.Nil(t, err, "CreateMnemonic err", err)

	addr24, err := w.Recovery(ct, mnemonic24, passphrase)
	assert.Nil(t, err, "recover addr by mnemonic err", err)

	err = w.Delete(addr24)
	assert.Nil(t, err, "delete addr err", err)

	err = removeTestWalletFolder(w, filepath.Join(testFolderPath(), "wallet"))
	assert.Nil(t, err, err)
}

//return path string of the folder which this file is in.
func testFolderPath() string {
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

func getTestEncrytWayInstance_ed25519(t *testing.T) EncryptWayOfWallet {
	var c ed25519.CryptServiceEd25519
	sec, pub, err := c.GeneratePriPubKey()
	assert.Nil(t, err, err)
	return EncryptWayOfWallet{
		CryptType: tpcrtypes.CryptType_Ed25519,
		Pubkey:    pub,
		Seckey:    sec,
	}
}

func getTestEncrytWayInstance_secp256(t *testing.T) EncryptWayOfWallet {
	var c secp256.CryptServiceSecp256
	sec, pub, err := c.GeneratePriPubKey()
	assert.Nil(t, err, err)
	return EncryptWayOfWallet{
		CryptType: tpcrtypes.CryptType_Secp256,
		Pubkey:    pub,
		Seckey:    sec,
	}
}

// removeTestWalletFolder just for Test.
func removeTestWalletFolder(w Wallet, path string) error {
	if !isValidFolderPath(path) {
		return errors.New("invalid folder path:" + path)
	}

	wImp := w.(*wallet)

	if fks, ok := wImp.ks.(*fileKeyStore); ok {
		fks.mutex.Lock()
		defaultAddrFile_mutex.Lock()
		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
		defaultAddrFile_mutex.Unlock()
		fks.mutex.Unlock()
	} else if kri, ok := wImp.ks.(*keyringImp); ok {
		kri.mutex.Lock()
		defaultAddrFile_mutex.Lock()
		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
		defaultAddrFile_mutex.Unlock()
		kri.mutex.Unlock()
	} else {
		return errors.New("wallet imp err")
	}
	return nil
}
