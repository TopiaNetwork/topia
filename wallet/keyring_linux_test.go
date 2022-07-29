//go:build linux
// +build linux

package wallet

import (
	"fmt"
	"github.com/99designs/keyring"
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

			var kri keyringImp
			err := initWithBackendX(&kri, keyring.KWalletBackend, testFolderPath())
			assert.Equal(t, nil, err, "init with backend:", keyring.KWalletBackend, "err:", err)
			testSetGetRemove(kri, t)

			err = os.RemoveAll(filepath.Join(testFolderPath(), topiaKeysFolderName))
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

			var kri keyringImp
			err := initWithBackendX(&kri, keyring.KeyCtlBackend, testFolderPath())
			assert.Nil(t, err, "init with KeyCtlBackend err", err)
			testSetGetRemove(kri, t)

			err = os.Remove(filepath.Join(testFolderPath(), topiaKeysFolderName))
			assert.Nil(t, err, "remove wallet folder err", err)
		}
	}

	if !match {
		fmt.Println("keyCtl isn't available yet, skip this test.")
	}
}
