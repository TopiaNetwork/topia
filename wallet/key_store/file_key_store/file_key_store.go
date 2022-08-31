package file_key_store

import (
	"encoding/json"
	"errors"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/wallet/cache"
	"github.com/TopiaNetwork/topia/wallet/key_store"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type EncryptWay struct {
	CryptType tpcrtypes.CryptType
	Pubkey    tpcrtypes.PublicKey
	Seckey    tpcrtypes.PrivateKey
}

type FileKeyStore struct {
	fileFolderPath string // path of the folder which contains keys
	mutex          sync.RWMutex
	EncryptWay
}

type InitArg struct {
	RootPath   string
	EncryptWay // For stored-message encryption and decryption
}

const keysFolderName = "wallet"
const lockFileName = "lock"

var (
	enableKeyNotSetErr = errors.New("enable key hasn't been set")
	addrNotExistErr    = errors.New("addr doesn't exist")
)

func (f *FileKeyStore) Init(arg InitArg) error {
	fileFolderPath := arg.RootPath
	if key_store.IsValidFolderPath(fileFolderPath) == false {
		return errors.New("input fileFolderPath is not a valid folder path")
	}

	keysFolderPath := filepath.Join(fileFolderPath, keysFolderName)
	err := os.Mkdir(keysFolderPath, 0544)
	if err != nil {
		if !os.IsExist(err) { // ignore dir already exist error.
			return err
		}
	}

	f.fileFolderPath = keysFolderPath
	f.EncryptWay = arg.EncryptWay

	err = f.createLockFile()
	if err != nil {
		return err
	}

	exist, err := key_store.IsPathExist(filepath.Join(f.fileFolderPath, key_store.EnableKey))
	if err != nil {
		return err
	}
	if !exist { // if walletEnableKey hasn't been set, set it
		err = f.SetEnable(true)
		if err != nil {
			return err
		}
	}

	err = loadKeysToCache(f)
	if err != nil {
		return err
	}

	return nil
}

func (f *FileKeyStore) SetAddr(addr string, item key_store.KeyItem) error {
	if len(addr) == 0 || item.CryptType == tpcrtypes.CryptType_Unknown || item.Seckey == nil {
		return errors.New("input invalid addrItem")
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	err := key_store.SetDirPerm(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))
	if err != nil {
		return err
	}
	defer key_store.SetDirReadOnly(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))

	err = f.lockEX()
	if err != nil {
		return err
	}
	defer f.unlock()

	bs, err := json.Marshal(item)
	if err != nil {
		return err
	}

	addrFilePath := filepath.Join(f.fileFolderPath, addr)

	exist, err := key_store.IsPathExist(addrFilePath)
	if err != nil {
		return err
	}

	var addrFile *os.File
	if !exist { // if addr doesn't exist, set it
		addrFile, err = os.Create(addrFilePath)
		if err != nil {
			return err
		}
		defer addrFile.Close()
	} else { // no err and addr exists
		addrFile, err = os.OpenFile(addrFilePath, os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer addrFile.Close()
	}

	encryptedData, err := key_store.StreamEncrypt(f.CryptType, f.Pubkey, bs)
	if err != nil {
		return err
	}
	_, err = addrFile.Write(encryptedData)
	if err != nil {
		return err
	}

	cache.SetAddrToCache(addr, encryptedData)
	return nil
}

func (f *FileKeyStore) GetAddr(addr string) (key_store.KeyItem, error) {
	if len(addr) == 0 {
		return key_store.KeyItem{}, errors.New("input invalid addr")
	}

	addrCacheItem, err := cache.GetAddrFromCache(addr)
	if err != nil { // didn't find addr in cache
		encKeyItem, err := f.getAddrItemFromBackend(addr)
		if err != nil {
			return key_store.KeyItem{}, err
		}
		cache.SetAddrToCache(addr, encKeyItem)
		addrCacheItem = cache.AddrItemInCache{
			EncKeyItem: encKeyItem, // lock: false
		}
	}

	decryptedMsg, err := key_store.StreamDecrypt(f.CryptType, f.Seckey, addrCacheItem.EncKeyItem)
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

func (f *FileKeyStore) SetEnable(set bool) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	err := key_store.SetDirPerm(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))
	if err != nil {
		return err
	}
	defer key_store.SetDirReadOnly(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))

	err = f.lockEX()
	if err != nil {
		return err
	}
	defer f.unlock()

	var temp []byte
	if set {
		temp = []byte(key_store.Enabled)
	} else {
		temp = []byte(key_store.Disabled)
	}

	weFilePath := filepath.Join(f.fileFolderPath, key_store.EnableKey)

	exist, err := key_store.IsPathExist(weFilePath)
	if err != nil {
		return err
	}

	var weFile *os.File
	if !exist { // don't find walletEnableKey then create it
		weFile, err = os.Create(weFilePath)
	} else {
		weFile, err = os.OpenFile(weFilePath, os.O_RDWR|os.O_TRUNC, 0644)
	}
	defer weFile.Close()

	if err != nil {
		return err
	}

	encryptedWE, err := key_store.StreamEncrypt(f.CryptType, f.Pubkey, temp)
	if err != nil {
		return err
	}
	_, err = weFile.Write(encryptedWE)
	if err != nil {
		return err
	}

	cache.SetEnableToCache(set)
	return nil
}

func (f *FileKeyStore) GetEnable() (bool, error) {
	return cache.GetEnableFromCache(), nil
}

func (f *FileKeyStore) Keys() (addrs []string, err error) {
	return cache.GetKeysFromCache(), nil
}

func (f *FileKeyStore) Remove(key string) error {
	if len(key) == 0 {
		return errors.New("input invalid addr")
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	err := key_store.SetDirPerm(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))
	if err != nil {
		return err
	}
	defer key_store.SetDirReadOnly(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))

	err = f.lockEX()
	if err != nil {
		return err
	}
	defer func(f *FileKeyStore) {
		err := f.unlock()
		if err != nil {

		}
	}(f)

	fp := filepath.Join(f.fileFolderPath, key)

	exist, err := key_store.IsPathExist(fp)
	if err != nil {
		return err
	}
	if !exist {
		return addrNotExistErr
	}

	err = os.Remove(fp)
	if err != nil {
		return err
	}

	cache.RemoveFromCache(key)

	return nil

}

// getAddrItemFromBackend return encrypted KeyItem from backend
func (f *FileKeyStore) getAddrItemFromBackend(addr string) (encKeyItem []byte, err error) {
	if len(addr) == 0 {
		return nil, errors.New("invalid addr")
	}

	f.mutex.RLock()
	defer f.mutex.RUnlock()

	err = f.lockSH()
	if err != nil {
		return nil, err
	}
	defer f.unlock()

	addrFilePath := filepath.Join(f.fileFolderPath, addr)
	exist, err := key_store.IsPathExist(addrFilePath)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, addrNotExistErr
	}

	data, err := ioutil.ReadFile(addrFilePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (f *FileKeyStore) getEnableFromBackend() (bool, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	err := f.lockSH()
	if err != nil {
		return false, err
	}
	defer f.unlock()

	weFile := filepath.Join(f.fileFolderPath, key_store.EnableKey)

	exist, err := key_store.IsPathExist(weFile)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, addrNotExistErr
	}

	we, err := ioutil.ReadFile(weFile)
	if err != nil {
		return false, err
	}

	decryptedWE, err := key_store.StreamDecrypt(f.CryptType, f.Seckey, we)
	if err != nil {
		return false, err
	}

	if string(decryptedWE) == key_store.Enabled {
		return true, nil
	} else if string(decryptedWE) == key_store.Disabled {
		return false, nil
	} else {
		return false, enableKeyNotSetErr
	}
}

func (f *FileKeyStore) listAddrsFromBackend() (addrs []string, err error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	err = f.lockSH()
	if err != nil {
		return nil, err
	}
	defer f.unlock()

	files, err := ioutil.ReadDir(f.fileFolderPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() { // skip dir
			continue
		}
		if !key_store.IsValidTopiaAddr(tpcrtypes.Address(file.Name())) { // skip irrelevant file
			continue
		}
		addrs = append(addrs, file.Name())
	}
	return addrs, nil
}

func (f *FileKeyStore) createLockFile() error {
	exist, err := key_store.IsPathExist(filepath.Join(f.fileFolderPath, lockFileName))
	if err != nil {
		return err
	}
	if exist {
		return nil
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	err = key_store.SetDirPerm(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))
	if err != nil {
		return err
	}
	defer key_store.SetDirReadOnly(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))

	file, err := os.Create(filepath.Join(f.fileFolderPath, lockFileName))
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

func loadKeysToCache(store *FileKeyStore) error {
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
