package wallet

import (
	"encoding/json"
	"errors"
	"github.com/99designs/keyring"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"os"
	"path/filepath"
	"sync"
)

/*
Item is a thing stored on the keyring.
type Item struct {
	Key         string --- to store Addr
	Data        []byte --- to store keyItem
	Label       string
	Description string
}
*/

type keyringImp struct {
	k     keyring.Keyring
	mutex sync.RWMutex // protect k
}

type keyringInitArg struct {
	EncryptWayOfWallet

	RootPath string
	Backend  string
}

const (
	keyringKeysFolderName = "wallet"
)

var _ keyStore = (*keyringImp)(nil)

func isValidFolderPath(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	if s.IsDir() == false {
		return false
	}
	return true
}

func (ki *keyringImp) Init(arg interface{}) error {
	initArgument, ok := arg.(keyringInitArg)
	if !ok {
		return errors.New("don't support your init argument type")
	}
	fileFolderPath := initArgument.RootPath
	if isValidFolderPath(fileFolderPath) == false {
		return errors.New("input fileFolderPath is not a valid folder path")
	}
	var bkd keyring.BackendType
	switch initArgument.Backend {
	case "file":
		bkd = keyring.FileBackend
	case "keychain":
		bkd = keyring.KeychainBackend
	case "keyctl":
		bkd = keyring.KeyCtlBackend
	case "kwallet":
		bkd = keyring.KWalletBackend
	case "wincred":
		bkd = keyring.WinCredBackend
	default:
		return errors.New("unsupported keyring backend")
	}

	config := keyring.Config{
		AllowedBackends:                []keyring.BackendType{bkd},
		ServiceName:                    "TopiaWalletStore",
		KeychainName:                   filepath.Join(fileFolderPath, keyringKeysFolderName, "TopiaWallet"),
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         true,
		KeychainAccessibleWhenUnlocked: true,
		FilePasswordFunc:               keyring.FixedStringPrompt(string(initArgument.Seckey)),
		FileDir:                        filepath.Join(fileFolderPath, keyringKeysFolderName),
		KeyCtlScope:                    "thread",
		KeyCtlPerm:                     0x3f3f0000, // "alswrvalswrv------------",
		KWalletAppID:                   "TopiaKWalletApp",
		KWalletFolder:                  "TopiaKWallet",
		WinCredPrefix:                  "", // default "keyring"
	}

	tempKeyring, err := keyring.Open(config)
	if err != nil {
		return errors.New("open keyring err: " + err.Error())
	}
	ki.k = tempKeyring

	if _, err = ki.getEnableFromBackend(); err != nil { // if walletEnableKey hasn't been set, set it
		err = ki.SetEnable(true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ki *keyringImp) SetAddr(addr string, item keyItem) error {
	if len(addr) == 0 || item.CryptType == tpcrtypes.CryptType_Unknown || item.Seckey == nil {
		return errors.New("input invalid addrItem")
	}

	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	dataItem := keyItem{
		Seckey:    item.Seckey,
		CryptType: item.CryptType,
	}
	bs, err := json.Marshal(dataItem)
	if err != nil {
		return err
	}
	keyringItem := keyring.Item{
		Key:  addr,
		Data: bs,
	}
	err = ki.k.Set(keyringItem)
	if err != nil {
		return err
	}

	cacheMutex.Lock()
	addr_Cache[addr] = addrItemInCache{
		keyItem: item,
		lock:    false,
	}
	cacheMutex.Unlock()

	return nil
}

func (ki *keyringImp) GetAddr(addr string) (keyItem, error) {
	if len(addr) == 0 {
		return keyItem{}, errors.New("input invalid addr")
	}

	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	ret, ok := addr_Cache[addr]
	if !ok {
		return keyItem{}, addrNotExistErr
	}
	return ret.keyItem, nil
}

func (ki *keyringImp) SetEnable(set bool) error {
	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	var temp []byte
	if set == true {
		temp = []byte(walletEnabled)
	} else {
		temp = []byte(walletDisabled)
	}

	item, err := ki.k.Get(walletEnableKey)
	if err != nil {
		item = keyring.Item{
			Key:  walletEnableKey,
			Data: temp,
		}
		return ki.k.Set(item)
	}
	item.Data = temp

	err = ki.k.Set(item)
	if err != nil {
		return err
	}

	cacheMutex.Lock()
	enable_Cache = set
	cacheMutex.Unlock()

	return nil
}

func (ki *keyringImp) GetEnable() (bool, error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	return enable_Cache, nil
}

func (ki *keyringImp) Keys() (addrs []string, err error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	for k := range addr_Cache {
		addrs = append(addrs, k)
	}

	return addrs, nil
}

func (ki *keyringImp) Remove(key string) error {
	if len(key) == 0 {
		return errors.New("input invalid addr")
	}

	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	err := ki.k.Remove(key)
	if err != nil {
		return err
	}

	cacheMutex.Lock()
	if key == walletEnableKey {
		enable_Cache = false
	} else {
		delete(addr_Cache, key)
	}
	cacheMutex.Unlock()

	return nil
}

func (ki *keyringImp) getAddrItemFromBackend(addr string) (keyItem, error) {
	if len(addr) == 0 {
		return keyItem{}, errors.New("input invalid addr")
	}

	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	item, err := ki.k.Get(addr)
	if err != nil {
		return keyItem{}, err
	}

	var addrItem keyItem
	err = json.Unmarshal(item.Data, &addrItem)
	if err != nil {
		return keyItem{}, err
	}

	return addrItem, nil
}

func (ki *keyringImp) getEnableFromBackend() (bool, error) {
	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	item, err := ki.k.Get(walletEnableKey)
	if err != nil {
		return false, err
	}

	if string(item.Data) == walletEnabled {
		return true, nil
	} else if string(item.Data) == walletDisabled {
		return false, nil
	} else {
		return false, errors.New("unexpected wallet enable state")
	}
}

func (ki *keyringImp) listAddrsFromBackend() (addrs []string, err error) {
	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	keys, err := ki.k.Keys()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if !isValidTopiaAddr(tpcrtypes.Address(key)) { // skip irrelevant file
			continue
		}
		addrs = append(addrs, key)
	}
	return addrs, nil
}
