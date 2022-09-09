package file_key_store

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
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
	cs             crypt.CryptService
	log            tplog.Logger

	EncryptWay
}

type InitArg struct {
	RootPath string
	Cs       crypt.CryptService
	Log      tplog.Logger

	EncryptWay // For stored-message encryption and decryption
}

const keysFolderName = "wallet"
const pidFileName = "pid"

var (
	enableKeyNotSetErr   = errors.New("enable key hasn't been set")
	defaultAddrNotSetErr = errors.New("default addr hasn't been set")
	addrNotExistErr      = errors.New("addr doesn't exist")
)

var fksInstance FileKeyStore

func InitStoreInstance(arg InitArg) (ks key_store.KeyStore, err error) {
	err = fksInstance.Init(arg)
	if err != nil {
		return nil, err
	}
	return &fksInstance, nil
}

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

	f.mutex.Lock()
	f.fileFolderPath = keysFolderPath
	f.cs = arg.Cs
	f.log = arg.Log
	f.EncryptWay = arg.EncryptWay
	f.mutex.Unlock()

	err = f.checkPidFile()
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

	exist, err = key_store.IsPathExist(filepath.Join(f.fileFolderPath, key_store.DefaultAddrKey))
	if err != nil {
		return err
	}
	if !exist { // if DefaultAddr hasn't been set, set it
		err = f.SetDefaultAddr("please_set_your_default_address")
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

	encryptedData, err := f.cs.StreamEncrypt(f.Pubkey, bs)
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

	decryptedMsg, err := f.cs.StreamDecrypt(f.Seckey, addrCacheItem.EncKeyItem)
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

	encryptedWE, err := f.cs.StreamEncrypt(f.Pubkey, temp)
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

func (f *FileKeyStore) SetDefaultAddr(defaultAddr string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	err := key_store.SetDirPerm(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))
	if err != nil {
		return err
	}
	defer key_store.SetDirReadOnly(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))

	daFilePath := filepath.Join(f.fileFolderPath, key_store.DefaultAddrKey)

	exist, err := key_store.IsPathExist(daFilePath)
	if err != nil {
		return err
	}
	var daFile *os.File
	if !exist { // don't find defaultAddrKey then create it
		daFile, err = os.Create(daFilePath)
		if err != nil {
			return err
		}
	} else {
		daFile, err = os.OpenFile(daFilePath, os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
	}
	defer daFile.Close()

	encryptedDA, err := f.cs.StreamEncrypt(f.Pubkey, []byte(defaultAddr))
	if err != nil {
		return err
	}
	_, err = daFile.Write(encryptedDA)
	if err != nil {
		return err
	}

	cache.SetDefaultAddrToCache(defaultAddr)
	return nil
}

func (f *FileKeyStore) GetDefaultAddr() (defaultAddr string, err error) {
	return cache.GetDefaultAddrFromCache(), nil
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

	decryptedWE, err := f.cs.StreamDecrypt(f.Seckey, we)
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

func (f *FileKeyStore) getDefaultAddrFromBackend() (defaultAddr string, err error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	daFilePath := filepath.Join(f.fileFolderPath, key_store.DefaultAddrKey)

	exist, err := key_store.IsPathExist(daFilePath)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", defaultAddrNotSetErr
	} else {
		bs, err := ioutil.ReadFile(daFilePath)
		if err != nil {
			return "", err
		}

		decryptedDA, err := f.cs.StreamDecrypt(f.Seckey, bs)
		if err != nil {
			return "", err
		}
		return string(decryptedDA), nil
	}
}

func (f *FileKeyStore) listAddrsFromBackend() (addrs []string, err error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

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

func (f *FileKeyStore) checkPidFile() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	err := key_store.SetDirPerm(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))
	if err != nil {
		return err
	}
	defer key_store.SetDirReadOnly(filepath.Dir(f.fileFolderPath), filepath.Base(f.fileFolderPath))

	pidFilePath := filepath.Join(f.fileFolderPath, pidFileName)
	exist, err := key_store.IsPathExist(pidFilePath)
	if err != nil {
		return err
	}

	pid := os.Getpid()
	var file *os.File
	if !exist { // if pidFile doesn't exist, set it
		file, err = os.Create(pidFilePath)
		if err != nil {
			return err
		}
		defer file.Close()

		pidBytes, err := intToBytes(pid)
		if err != nil {
			return err
		}
		_, err = file.Write(pidBytes)
		if err != nil {
			return err
		}
		return nil

	} else { // no err and pidFile exists
		pidBytes, err := ioutil.ReadFile(pidFilePath)
		if err != nil {
			return err
		}
		pidInLock, err := bytesToInt(pidBytes)
		if err != nil {
			return err
		}
		pidAlive, err := isPIDAlive(pidInLock)
		if err != nil {
			return err
		}
		if pidAlive {
			f.log.Errorf("pid: %d is running. Wallet exits.", pidInLock)
			os.Exit(0)
		}

		file, err = os.OpenFile(pidFilePath, os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		myPidBytes, err := intToBytes(pid)
		if err != nil {
			return err
		}
		_, err = file.Write(myPidBytes)
		if err != nil {
			return err
		}
		return nil
	}
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

	defaultAddr, err := store.getDefaultAddrFromBackend()
	if err != nil {
		cache.SetDefaultAddrToCache("")
		return err
	}
	cache.SetDefaultAddrToCache(defaultAddr)
	return nil
}

func intToBytes(n int) ([]byte, error) {
	data := int64(n)
	byteBuf := bytes.NewBuffer([]byte{})
	err := binary.Write(byteBuf, binary.LittleEndian, data)
	if err != nil {
		return nil, err
	}
	return byteBuf.Bytes(), nil
}

func bytesToInt(bys []byte) (int, error) {
	byteBuf := bytes.NewBuffer(bys)
	var data int64
	err := binary.Read(byteBuf, binary.LittleEndian, &data)
	if err != nil {
		return 0, err
	}
	return int(data), nil
}
