//go:build windows
// +build windows

package wallet

import (
	"github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// windows only
func TestKeyringWithBKD_Wincred(t *testing.T) {
	cleanCache()

	var kri keyringImp
	err := initWithBackendX(&kri, keyring.WinCredBackend, testFolderPath())
	assert.Equal(t, nil, err, "init with backend:", keyring.FileBackend, "err:", err)
	testSetGetRemove(kri, t)

	err = os.RemoveAll(filepath.Join(testFolderPath(), topiaKeysFolderName))
	assert.Nil(t, err, "remove wallet folder err", err)
}
