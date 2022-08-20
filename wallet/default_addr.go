package wallet

import (
	"errors"
	"github.com/TopiaNetwork/topia/wallet/key_store"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

const (
	keysFolderName = "wallet"
)

var (
	defaultAddrFile_mutex  sync.Mutex
	defaultAddrCache_mutex sync.RWMutex
)

var defaultAddr_Cache = "" // default void-string

var defaultAddrNotSetErr = errors.New("default addr hasn't been set")

func initDefaultAddr() error {
	defaultAddrFolderPath := filepath.Join(walletBackendConfig.RootPath, keysFolderName)
	exist, err := key_store.IsPathExist(filepath.Join(defaultAddrFolderPath, key_store.DefaultAddrKey))
	if err != nil {
		return err
	}
	if !exist { // if walletDefaultAddrKey hasn't been set, set it
		return SetDefaultAddr("")
	}

	defaultAddr, err := getDefaultAddrFromBackend()
	if err != nil {
		return err
	}
	defaultAddrCache_mutex.Lock()
	defaultAddr_Cache = defaultAddr
	defaultAddrCache_mutex.Unlock()
	return nil
}

func SetDefaultAddr(defaultAddr string) error {
	defaultAddrFile_mutex.Lock()
	defer defaultAddrFile_mutex.Unlock()

	keysFolderPath := filepath.Join(walletBackendConfig.RootPath, keysFolderName)
	err := os.Mkdir(keysFolderPath, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) { // ignore dir already exist error.
			return err
		}
	}

	daFilePath := filepath.Join(keysFolderPath, key_store.DefaultAddrKey)

	exist, err := key_store.IsPathExist(daFilePath)
	if err != nil {
		return err
	}
	var daFile *os.File
	if !exist { // don't find defaultAddrKey then create it
		daFile, err = os.Create(daFilePath)
	} else {
		daFile, err = os.OpenFile(daFilePath, os.O_RDWR|os.O_TRUNC, 0644)
	}
	defer daFile.Close()
	if err != nil {
		return err
	}

	_, err = daFile.Write([]byte(defaultAddr))
	if err != nil {
		return err
	}

	defaultAddrCache_mutex.Lock()
	defaultAddr_Cache = defaultAddr
	defaultAddrCache_mutex.Unlock()
	return nil
}

func GetDefaultAddr() (defaultAddr string) {
	defaultAddrCache_mutex.RLock()
	defaultAddr = defaultAddr_Cache
	defaultAddrCache_mutex.RUnlock()
	return defaultAddr
}

func RemoveDefaultAddr() error {
	defaultAddrFile_mutex.Lock()
	defer defaultAddrFile_mutex.Unlock()

	keysFolderPath := filepath.Join(walletBackendConfig.RootPath, keysFolderName)
	fp := filepath.Join(keysFolderPath, key_store.DefaultAddrKey)

	exist, err := key_store.IsPathExist(fp)
	if err != nil {
		return err
	}
	if !exist {
		return defaultAddrNotSetErr
	}

	err = os.Remove(fp)
	if err != nil {
		return err
	}

	defaultAddrCache_mutex.Lock()
	defaultAddr_Cache = "" // default void string
	defaultAddrCache_mutex.Unlock()

	return nil
}

func getDefaultAddrFromBackend() (defaultAddr string, err error) {
	defaultAddrFile_mutex.Lock()
	defer defaultAddrFile_mutex.Unlock()

	defaultAddrFolderPath := filepath.Join(walletBackendConfig.RootPath, keysFolderName)
	daFilePath := filepath.Join(defaultAddrFolderPath, key_store.DefaultAddrKey)

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
		return string(bs), nil
	}
}
