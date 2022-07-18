package wallet

import (
	"bytes"
	"encoding/gob"
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
	Data        []byte --- to store itemData
	Label       string
	Description string
}
*/
type itemData struct {
	Lock      bool
	Seckey    tpcrtypes.PrivateKey
	Pubkey    tpcrtypes.PublicKey
	CryptType tpcrtypes.CryptType
}

type keyringImp struct {
	k     keyring.Keyring
	mutex sync.RWMutex // protect k
}

type keyringInitArg struct {
	RootPath string
	Backend  string
}

const (
	keyringKeysFolderName = "wallet"
)

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
		KeychainPasswordFunc:           keyring.FixedStringPrompt("password"),
		FilePasswordFunc:               keyring.FixedStringPrompt("password"),
		FileDir:                        filepath.Join(fileFolderPath, keyringKeysFolderName),
		KeyCtlScope:                    "thread",
		KeyCtlPerm:                     0x3f3f0000, // "alswrvalswrv------------",
		KWalletAppID:                   "TopiaKWalletApp",
		KWalletFolder:                  "TopiaKWallet",
		WinCredPrefix:                  "", // default "keyring"
	}
	ki.mutex.Lock()
	defer ki.mutex.Unlock()
	tempKeyring, err := keyring.Open(config)
	if err != nil {
		return errors.New("open keyring err: " + err.Error())
	}
	ki.k = tempKeyring

	if _, err = ki.k.Get(walletEnableKey); err != nil { // if walletEnableKey hasn't been set, set it
		tempItem := keyring.Item{
			Key:  walletEnableKey,
			Data: []byte(walletEnabled),
		}
		if err = ki.k.Set(tempItem); err != nil {
			return errors.New("set walletEnable Item err: " + err.Error())
		}
	}

	if _, err = ki.k.Get(defaultAddrKey); err != nil { // if walletDefaultAddrKey hasn't been set, set it
		tempItem := keyring.Item{
			Key:  defaultAddrKey,
			Data: []byte("please_set_default_address"),
		}
		if err = ki.k.Set(tempItem); err != nil {
			return errors.New("set walletDefaultAddr Item err: " + err.Error())
		}
	}

	return nil
}

func (ki *keyringImp) SetAddr(item addrItem) error {
	if len(item.Addr) == 0 || item.CryptType == tpcrtypes.CryptType_Unknown || item.Seckey == nil || item.Pubkey == nil {
		return errors.New("input invalid addrItem")
	}

	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	dataItem := itemData{
		Lock:      item.AddrLocked,
		Seckey:    item.Seckey,
		Pubkey:    item.Pubkey,
		CryptType: item.CryptType,
	}
	bs, err := itemDataToByteSlice(dataItem)
	if err != nil {
		return err
	}
	keyringItem := keyring.Item{
		Key:  item.Addr,
		Data: bs,
	}
	err = ki.k.Set(keyringItem)
	if err != nil {
		return err
	}
	return nil
}

func (ki *keyringImp) GetAddr(addr string) (addrItem, error) {
	if len(addr) == 0 {
		return addrItem{}, errors.New("input invalid addr")
	}

	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	item, err := ki.k.Get(addr)
	if err != nil {
		return addrItem{}, err
	}
	dataItem, err := byteSliceToItemData(item.Data)
	if err != nil {
		return addrItem{}, err
	}
	return addrItem{
		Addr:       item.Key,
		AddrLocked: dataItem.Lock,
		Seckey:     dataItem.Seckey,
		Pubkey:     dataItem.Pubkey,
		CryptType:  dataItem.CryptType,
	}, nil
}

func (ki *keyringImp) RemoveItem(key string) error {
	if len(key) == 0 {
		return errors.New("input invalid addr")
	}

	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	return ki.k.Remove(key)
}

func (ki *keyringImp) List() (addrs []string, err error) {
	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	keys, err := ki.k.Keys()
	if err != nil {
		return nil, err
	}

	totalNum := len(keys)
	for i := 0; i < totalNum; i++ {
		if keys[i] == walletEnableKey || keys[i] == defaultAddrKey {
			keys = append(keys[:i], keys[i+1:]...)
			totalNum--
			i--
		}
	}

	return keys, nil
}

func (ki *keyringImp) SetWalletEnable(set bool) error {
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
	return ki.k.Set(item)
}

func (ki *keyringImp) GetWalletEnable() (bool, error) {
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

func (ki *keyringImp) SetDefaultAddr(defaultAddr string) error {
	if len(defaultAddr) == 0 {
		return errors.New("input invalid addr")
	}
	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	item, err := ki.k.Get(defaultAddrKey)
	if err != nil {
		item = keyring.Item{
			Key:  defaultAddrKey,
			Data: []byte(defaultAddr),
		}
		return ki.k.Set(item)
	}
	item.Data = []byte(defaultAddr)
	return ki.k.Set(item)
}

func (ki *keyringImp) GetDefaultAddr() (defaultAddr string, err error) {
	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	item, err := ki.k.Get(defaultAddrKey)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func byteSliceToItemData(data []byte) (ret itemData, err error) {
	if data == nil {
		return ret, errors.New("input nil data")
	}

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func itemDataToByteSlice(data itemData) (ret []byte, err error) {
	if data.Seckey == nil || data.Pubkey == nil || data.CryptType == tpcrtypes.CryptType_Unknown {
		return nil, errors.New("input invalid itemData")
	}

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err = enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
