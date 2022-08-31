//go:build windows
// +build windows

package keyring

import (
	"github.com/99designs/keyring"
	"github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// windows only
func TestKeyringWithBKD_Wincred(t *testing.T) {
	var kri KeyringImp
	initArg := InitArg{
		EncryptWay: getTestEncrytWayInstance_secp256(t),
		RootPath:   dirPathForTest(),
		Backend:    string(keyring.WinCredBackend),
		Cs:         crypt.CreateCryptService(nil, tpcrtypes.CryptType_Secp256),
	}
	err := kri.Init(initArg)
	assert.Equal(t, nil, err, "init with backend:", keyring.WinCredBackend, "err:", err)
	testSetGetRemove(&kri, t)

	err = os.RemoveAll(filepath.Join(dirPathForTest(), keysFolderName))
	assert.Nil(t, err, "remove wallet folder err", err)
}
