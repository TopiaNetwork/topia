//go:build darwin

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

// macOS only
func TestKeyringWithBKD_Keychain(t *testing.T) {
	var kri KeyringImp
	initArg := InitArg{
		EncryptWay: getTestEncrytWayInstance_ed25519(t),
		RootPath:   dirPathForTest(),
		Backend:    string(keyring.KeychainBackend),
		Cs:         crypt.CreateCryptService(nil, tpcrtypes.CryptType_Ed25519),
	}
	err := kri.Init(initArg)
	assert.Equal(t, nil, err, "init with backend:", keyring.KeychainBackend, "err:", err)
	testSetGetRemove(&kri, t)

	err = os.RemoveAll(filepath.Join(dirPathForTest(), keysFolderName))
	assert.Nil(t, err, "remove wallet folder err", err)
}
