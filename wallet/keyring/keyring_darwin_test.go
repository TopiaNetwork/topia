//go:build darwin
// +build darwin

package keyring

import (
	"github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// macOS only
func TestKeyringWithBKD_Keychain(t *testing.T) {
	var kri KeyringImp
	err := initWithBackendX(&kri, keyring.KeychainBackend, dirPathForTest(), getTestEncrytWayInstance_secp256(t))
	assert.Equal(t, nil, err, "init with backend:", keyring.FileBackend, "err:", err)
	testSetGetRemove(&kri, t)

	err = os.RemoveAll(filepath.Join(dirPathForTest(), keysFolderName))
	assert.Nil(t, err, "remove wallet folder err", err)
}
