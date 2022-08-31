package keyring

import (
	"encoding/json"
	"errors"
	"github.com/99designs/keyring"
	"github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/wallet/cache"
	"github.com/TopiaNetwork/topia/wallet/key_store"
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

type EncryptWay struct {
	CryptType tpcrtypes.CryptType
	Pubkey    tpcrtypes.PublicKey
	Seckey    tpcrtypes.PrivateKey
}

type KeyringImp struct {
	EncryptWay

	k     keyring.Keyring
	mutex sync.RWMutex // protect k
	cs    crypt.CryptService
}

type InitArg struct {
	EncryptWay

	RootPath string
	Backend  string
	Cs       crypt.CryptService
}

const (
	keysFolderName = "wallet"

	serviceName          = "TopiaWalletStore"
	keyCtlScope          = "thread"
	keyCtlPerm    uint32 = 0x3f3f0000 // "alswrvalswrv------------"
	kWalletAppID         = "TopiaKWalletApp"
	kWalletFolder        = "TopiaKWallet"
	winCredPrefix        = "" // default "keyring"
)

var _ key_store.KeyStore = (*KeyringImp)(nil)

var kriInstance KeyringImp

func InitStoreInstance(arg InitArg) (ks key_store.KeyStore, err error) {
	err = kriInstance.Init(arg)
	if err != nil {
		return nil, err
	}
	return &kriInstance, nil
}

func (ki *KeyringImp) Init(arg InitArg) error {
	fileFolderPath := arg.RootPath
	if key_store.IsValidFolderPath(fileFolderPath) == false {
		return errors.New("input fileFolderPath is not a valid folder path")
	}

	var bkd keyring.BackendType
	switch arg.Backend {
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
		panic("unsupported keyring backend")
	}

	config := keyring.Config{
		AllowedBackends:                []keyring.BackendType{bkd},
		ServiceName:                    serviceName,
		KeychainName:                   filepath.Join(fileFolderPath, keysFolderName, "TopiaWallet"),
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         true,
		KeychainAccessibleWhenUnlocked: true,
		FilePasswordFunc:               keyring.FixedStringPrompt(string(arg.Seckey)),
		FileDir:                        filepath.Join(fileFolderPath, keysFolderName),
		KeyCtlScope:                    keyCtlScope,
		KeyCtlPerm:                     keyCtlPerm,
		KWalletAppID:                   kWalletAppID,
		KWalletFolder:                  kWalletFolder,
		WinCredPrefix:                  winCredPrefix,
	}

	tempKeyring, err := keyring.Open(config)
	if err != nil {
		return errors.New("open keyring err: " + err.Error())
	}

	ki.mutex.Lock()
	ki.k = tempKeyring
	ki.EncryptWay = arg.EncryptWay
	ki.cs = arg.Cs
	ki.mutex.Unlock()

	if _, err = ki.getEnableFromBackend(); err != nil { // if walletEnableKey hasn't been set, set it
		err = ki.SetEnable(true)
		if err != nil {
			return err
		}
	}

	err = loadKeysToCache(ki)
	if err != nil {
		return err
	}

	return nil
}

func (ki *KeyringImp) SetAddr(addr string, item key_store.KeyItem) error {
	if len(addr) == 0 || item.CryptType == tpcrtypes.CryptType_Unknown || item.Seckey == nil {
		return errors.New("input invalid addrItem")
	}

	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	dataItem := key_store.KeyItem{
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

	encryptedData, err := ki.cs.StreamEncrypt(ki.Pubkey, bs)
	if err != nil {
		return err
	}

	cache.SetAddrToCache(addr, encryptedData)

	return nil
}

func (ki *KeyringImp) GetAddr(addr string) (key_store.KeyItem, error) {
	if len(addr) == 0 {
		return key_store.KeyItem{}, errors.New("input invalid addr")
	}

	addrCacheItem, err := cache.GetAddrFromCache(addr)
	if err != nil {
		encKeyItem, err := ki.getAddrItemFromBackend(addr)
		if err != nil {
			return key_store.KeyItem{}, err
		}
		cache.SetAddrToCache(addr, encKeyItem)
		addrCacheItem = cache.AddrItemInCache{
			EncKeyItem: encKeyItem,
		}
	}

	decryptedMsg, err := ki.cs.StreamDecrypt(ki.Seckey, addrCacheItem.EncKeyItem)
	if err != nil {
		return key_store.KeyItem{}, err
	}

	var ret key_store.KeyItem
	err = json.Unmarshal(decryptedMsg, &ret)
	if err != nil {
		return key_store.KeyItem{}, err
	}

	return ret, nil
}

func (ki *KeyringImp) SetEnable(set bool) error {
	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	var temp []byte
	if set == true {
		temp = []byte(key_store.Enabled)
	} else {
		temp = []byte(key_store.Disabled)
	}

	item, err := ki.k.Get(key_store.EnableKey)
	if err != nil {
		item = keyring.Item{
			Key:  key_store.EnableKey,
			Data: temp,
		}
		return ki.k.Set(item)
	}
	item.Data = temp

	err = ki.k.Set(item)
	if err != nil {
		return err
	}

	cache.SetEnableToCache(set)
	return nil
}

func (ki *KeyringImp) GetEnable() (bool, error) {
	return cache.GetEnableFromCache(), nil
}

func (ki *KeyringImp) Keys() (addrs []string, err error) {
	return cache.GetKeysFromCache(), nil
}

func (ki *KeyringImp) Remove(key string) error {
	if len(key) == 0 {
		return errors.New("input invalid addr")
	}

	ki.mutex.Lock()
	defer ki.mutex.Unlock()

	err := ki.k.Remove(key)
	if err != nil {
		return err
	}

	cache.RemoveFromCache(key)
	return nil
}

// getAddrItemFromBackend return encrypted KeyItem from backend
func (ki *KeyringImp) getAddrItemFromBackend(addr string) (encKeyItem []byte, err error) {
	if len(addr) == 0 {
		return nil, errors.New("input invalid addr")
	}

	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	item, err := ki.k.Get(addr)
	if err != nil {
		return nil, err
	}

	encryptedData, err := ki.cs.StreamEncrypt(ki.Pubkey, item.Data)
	if err != nil {
		return nil, err
	}

	return encryptedData, nil
}

func (ki *KeyringImp) getEnableFromBackend() (bool, error) {
	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	item, err := ki.k.Get(key_store.EnableKey)
	if err != nil {
		return false, err
	}

	if string(item.Data) == key_store.Enabled {
		return true, nil
	} else if string(item.Data) == key_store.Disabled {
		return false, nil
	} else {
		return false, errors.New("unexpected wallet enable state")
	}
}

func (ki *KeyringImp) listAddrsFromBackend() (addrs []string, err error) {
	ki.mutex.RLock()
	defer ki.mutex.RUnlock()

	keys, err := ki.k.Keys()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if !key_store.IsValidTopiaAddr(tpcrtypes.Address(key)) { // skip irrelevant file
			continue
		}
		addrs = append(addrs, key)
	}
	return addrs, nil
}

func loadKeysToCache(store *KeyringImp) error {
	addrs, err := store.listAddrsFromBackend()
	if err != nil {
		return err
	}

	for _, addr := range addrs { // synchronize backend addrs to addr_Cache
		_, err = cache.GetAddrFromCache(addr)
		if err != nil { // didn't find addr in cache
			item, err := store.getAddrItemFromBackend(addr)
			if err != nil {
				return err
			}
			cache.SetAddrToCache(addr, item)
		}
	}

	enable, err := store.getEnableFromBackend()
	if err != nil {
		cache.SetEnableToCache(false)
		return err
	}
	cache.SetEnableToCache(enable)
	return nil
}
