//go:build darwin
// +build darwin

package wallet

import (
	"github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"testing"
)

// macOS only
func TestKeyringWithBKD_Keychain(t *testing.T) {
	var kri keyringImp
	err := initWithBackendX(&kri, keyring.KeychainBackend, testFolderPath())
	assert.Equal(t, nil, err, "init with backend:", keyring.FileBackend, "err:", err)
	testSetGetRemove(kri, t)

	err = os.RemoveAll(filepath.Join(testFolderPath(), topiaKeysFolderName))
	assert.Nil(t, err, "remove wallet folder err", err)
}
