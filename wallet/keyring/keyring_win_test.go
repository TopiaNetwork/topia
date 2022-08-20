//go:build windows
// +build windows

package keyring

import (
	"github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// windows only
func TestKeyringWithBKD_Wincred(t *testing.T) {
	var kri KeyringImp
	err := initWithBackendX(&kri, keyring.WinCredBackend, dirPathForTest(), getTestEncrytWayInstance_secp256(t))
	assert.Equal(t, nil, err, "init with backend:", keyring.FileBackend, "err:", err)
	testSetGetRemove(&kri, t)

	err = os.RemoveAll(filepath.Join(dirPathForTest(), keysFolderName))
	assert.Nil(t, err, "remove wallet folder err", err)
}
