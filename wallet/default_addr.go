package wallet

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var defaultAddrFile_mutex sync.Mutex

func initDefaultAddrFile() error {
	defaultAddrFolderPath := filepath.Join(walletBackendConfig.RootPath, topiaKeysFolderName)
	exist, err := isPathExist(filepath.Join(defaultAddrFolderPath, defaultAddrKey))
	if err != nil {
		return err
	}
	if !exist { // if walletDefaultAddrKey hasn't been set, set it
		return setDefaultAddr("")
	}
	return nil
}

func setDefaultAddr(defaultAddr string) error {
	defaultAddrFile_mutex.Lock()
	defer defaultAddrFile_mutex.Unlock()

	keysFolderPath := filepath.Join(walletBackendConfig.RootPath, topiaKeysFolderName)
	err := os.Mkdir(keysFolderPath, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) { // ignore dir already exist error.
			return err
		}
	}

	daFilePath := filepath.Join(keysFolderPath, defaultAddrKey)

	exist, err := isPathExist(daFilePath)
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

	cacheMutex.Lock()
	defaultAddr_Cache = defaultAddr
	cacheMutex.Unlock()
	return nil
}

func getDefaultAddr() (defaultAddr string) {
	cacheMutex.RLock()
	defaultAddr = defaultAddr_Cache
	cacheMutex.RUnlock()
	return defaultAddr
}

func removeDefaultAddr() error {
	defaultAddrFile_mutex.Lock()
	defer defaultAddrFile_mutex.Unlock()

	keysFolderPath := filepath.Join(walletBackendConfig.RootPath, topiaKeysFolderName)
	fp := filepath.Join(keysFolderPath, defaultAddrKey)

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
	defaultAddr_Cache = "" // default void string
	cacheMutex.Unlock()

	return nil
}

func getDefaultAddrFromBackend() (defaultAddr string, err error) {
	defaultAddrFile_mutex.Lock()
	defer defaultAddrFile_mutex.Unlock()

	defaultAddrFolderPath := filepath.Join(walletBackendConfig.RootPath, topiaKeysFolderName)
	daFilePath := filepath.Join(defaultAddrFolderPath, defaultAddrKey)

	exist, err := isPathExist(daFilePath)
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
