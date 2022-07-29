package wallet

import (
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/crypt"
	"github.com/TopiaNetwork/topia/crypt/ed25519"
	"github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type fileKeyStore struct {
	fileFolderPath string // path of the folder which contains keys
	mutex          sync.RWMutex
	EncryptWayOfWallet
}

type initArg struct {
	RootPath           string
	EncryptWayOfWallet // For stored-message encryption and decryption
}

const topiaKeysFolderName = "wallet"

var (
	enableKeyNotSetErr   = errors.New("enable key hasn't been set")
	defaultAddrNotSetErr = errors.New("default addr hasn't been set")
	addrNotExistErr      = errors.New("addr doesn't exist")
)

var _ keyStore = (*fileKeyStore)(nil)

func (f *fileKeyStore) Init(arg interface{}) error {
	initArgument, ok := arg.(initArg)
	if !ok {
		return errors.New("don't support your init argument type")
	}
	fileFolderPath := initArgument.RootPath
	if isValidFolderPath(fileFolderPath) == false {
		return errors.New("input fileFolderPath is not a valid folder path")
	}

	keysFolderPath := filepath.Join(fileFolderPath, topiaKeysFolderName)
	err := os.Mkdir(keysFolderPath, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) { // ignore dir already exist error.
			return err
		}
	}

	f.fileFolderPath = keysFolderPath
	f.EncryptWayOfWallet = initArgument.EncryptWayOfWallet

	exist, err := isPathExist(filepath.Join(f.fileFolderPath, walletEnableKey))
	if err != nil {
		return err
	}
	if !exist { // if walletEnableKey hasn't been set, set it
		err = f.SetEnable(true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *fileKeyStore) SetAddr(addr string, item keyItem) error {
	if len(addr) == 0 || item.CryptType == tpcrtypes.CryptType_Unknown || item.Seckey == nil {
		return errors.New("input invalid addrItem")
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	bs, err := json.Marshal(item)
	if err != nil {
		return err
	}

	addrFilePath := filepath.Join(f.fileFolderPath, addr)

	exist, err := isPathExist(addrFilePath)
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
	var c crypt.CryptService
	switch f.CryptType {
	case tpcrtypes.CryptType_Secp256:
		c = new(secp256.CryptServiceSecp256)
	case tpcrtypes.CryptType_Ed25519:
		c = new(ed25519.CryptServiceEd25519)
	default:
		return errors.New("unsupported CryptType")
	}

	encryptedData, err := c.StreamEncrypt(f.Pubkey, bs)
	if err != nil {
		return err
	}
	_, err = addrFile.Write(encryptedData)
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

func (f *fileKeyStore) GetAddr(addr string) (keyItem, error) {
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

func (f *fileKeyStore) SetEnable(set bool) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var temp []byte
	if set {
		temp = []byte(walletEnabled)
	} else {
		temp = []byte(walletDisabled)
	}

	weFilePath := filepath.Join(f.fileFolderPath, walletEnableKey)

	exist, err := isPathExist(weFilePath)
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

	var c crypt.CryptService
	switch f.CryptType {
	case tpcrtypes.CryptType_Secp256:
		c = new(secp256.CryptServiceSecp256)
	case tpcrtypes.CryptType_Ed25519:
		c = new(ed25519.CryptServiceEd25519)
	default:
		return errors.New("unsupported CryptType")
	}
	encryptedWE, err := c.StreamEncrypt(f.Pubkey, temp)
	if err != nil {
		return err
	}
	_, err = weFile.Write(encryptedWE)
	if err != nil {
		return err
	}

	cacheMutex.Lock()
	enable_Cache = set
	cacheMutex.Unlock()
	return nil
}

func (f *fileKeyStore) GetEnable() (bool, error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	return enable_Cache, nil
}

func (f *fileKeyStore) Keys() (addrs []string, err error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	for k := range addr_Cache {
		addrs = append(addrs, k)
	}

	return addrs, nil
}

func (f *fileKeyStore) Remove(key string) error {
	if len(key) == 0 {
		return errors.New("input invalid addr")
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	fp := filepath.Join(f.fileFolderPath, key)

	exist, err := isPathExist(fp)
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

	cacheMutex.Lock()
	if key == walletEnableKey {
		enable_Cache = false
	} else {
		delete(addr_Cache, key)
	}
	cacheMutex.Unlock()

	return nil

}

func (f *fileKeyStore) getAddrItemFromBackend(addr string) (keyItem, error) {
	if len(addr) == 0 {
		return keyItem{}, errors.New("invalid addr")
	}

	f.mutex.RLock()
	defer f.mutex.RUnlock()

	addrFilePath := filepath.Join(f.fileFolderPath, addr)
	exist, err := isPathExist(addrFilePath)
	if err != nil {
		return keyItem{}, err
	}
	if !exist {
		return keyItem{}, addrNotExistErr
	}

	data, err := ioutil.ReadFile(addrFilePath)
	if err != nil {
		return keyItem{}, err
	}

	var c crypt.CryptService
	switch f.CryptType {
	case tpcrtypes.CryptType_Secp256:
		c = new(secp256.CryptServiceSecp256)
	case tpcrtypes.CryptType_Ed25519:
		c = new(ed25519.CryptServiceEd25519)
	default:
		return keyItem{}, errors.New("unsupported CryptType")
	}
	decryptedAddr, err := c.StreamDecrypt(f.Seckey, data)
	if err != nil {
		return keyItem{}, err
	}

	var ret keyItem
	err = json.Unmarshal(decryptedAddr, &ret)
	if err != nil {
		return keyItem{}, err
	}
	return ret, nil
}

func (f *fileKeyStore) getEnableFromBackend() (bool, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	weFile := filepath.Join(f.fileFolderPath, walletEnableKey)

	exist, err := isPathExist(weFile)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, enableKeyNotSetErr
	}

	we, err := ioutil.ReadFile(weFile)
	if err != nil {
		return false, err
	}

	var c crypt.CryptService
	switch f.CryptType {
	case tpcrtypes.CryptType_Secp256:
		c = new(secp256.CryptServiceSecp256)
	case tpcrtypes.CryptType_Ed25519:
		c = new(ed25519.CryptServiceEd25519)
	default:
		return false, errors.New("unsupported CryptType")
	}
	decryptedWE, err := c.StreamDecrypt(f.Seckey, we)
	if err != nil {
		return false, err
	}

	if string(decryptedWE) == walletEnabled {
		return true, nil
	} else if string(decryptedWE) == walletDisabled {
		return false, nil
	} else {
		return false, enableKeyNotSetErr
	}
}

func (f *fileKeyStore) listAddrsFromBackend() (addrs []string, err error) {
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
		if !isValidTopiaAddr(tpcrtypes.Address(file.Name())) { // skip irrelevant file
			continue
		}
		addrs = append(addrs, file.Name())
	}
	return addrs, nil
}

func isPathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
