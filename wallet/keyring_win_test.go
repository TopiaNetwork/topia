//go:build windows
// +build windows

package wallet

import (
	"github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"testing"
)

// windows only
func TestKeyringWithBKD_Wincred(t *testing.T) {
	var kri keyringImp
	err := initWithBackendX(&kri, keyring.WinCredBackend, testFolderPath())
	assert.Equal(t, nil, err, "init with backend:", keyring.FileBackend, "err:", err)
	testSetGetRemove(kri, t)
}
