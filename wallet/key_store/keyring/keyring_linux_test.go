//go:build linux
// +build linux

package keyring

import (
	"fmt"
	"github.com/99designs/keyring"
	"github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// linux only
func TestKeyringWithBKD_KWallet(t *testing.T) {
	match := false
	for _, bkd := range keyring.AvailableBackends() {
		if bkd == keyring.KWalletBackend {
			match = true

			var kri KeyringImp
			initArg := InitArg{
				EncryptWay: getTestEncrytWayInstance_secp256(t),
				RootPath:   dirPathForTest(),
				Backend:    string(keyring.KWalletBackend),
				Cs:         crypt.CreateCryptService(nil, tpcrtypes.CryptType_Secp256),
			}
			err := kri.Init(initArg)
			assert.Equal(t, nil, err, "init with backend:", keyring.KWalletBackend, "err:", err)
			testSetGetRemove(&kri, t)

			err = os.RemoveAll(filepath.Join(dirPathForTest(), keysFolderName))
			assert.Nil(t, err, "remove wallet folder err", err)
		}
	}

	if !match {
		fmt.Println("KWallet isn't available yet, skip this test.")
	}
}

// linux only
func TestKeyringWithBKD_KeyCtl(t *testing.T) {
	match := false
	for _, bkd := range keyring.AvailableBackends() {
		if bkd == keyring.KeyCtlBackend {
			match = true

			var kri KeyringImp
			initArg := InitArg{
				EncryptWay: getTestEncrytWayInstance_secp256(t),
				RootPath:   dirPathForTest(),
				Backend:    string(keyring.KeyCtlBackend),
				Cs:         crypt.CreateCryptService(nil, tpcrtypes.CryptType_Secp256),
			}
			err := kri.Init(initArg)
			assert.Nil(t, err, "init with KeyCtlBackend err", err)
			testSetGetRemove(&kri, t)

			err = os.RemoveAll(filepath.Join(dirPathForTest(), keysFolderName))
			assert.Nil(t, err, "remove wallet folder err", err)
		}
	}

	if !match {
		fmt.Println("keyCtl isn't available yet, skip this test.")
	}
}
