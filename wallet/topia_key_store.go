package wallet

import (
	"bytes"
	"encoding/gob"
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
	walletEnableKeyNotSetErr = errors.New("wallet enable key hasn't been set yet")
	defaultAddrNotSetErr     = errors.New("default addr hasn't been set yet")
	addrNotExistErr          = errors.New("addr doesn't exist in wallet")
)

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
		err = f.SetWalletEnable(true)
		if err != nil {
			return err
		}
	}

	exist, err = isPathExist(filepath.Join(f.fileFolderPath, defaultAddrKey))
	if err != nil {
		return err
	}
	if !exist { // if walletDefaultAddrKey hasn't been set, set it
		err = f.SetDefaultAddr("please_set_your_default_addr")
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *fileKeyStore) SetAddr(item addrItem) error {
	if len(item.Addr) == 0 || item.CryptType == tpcrtypes.CryptType_Unknown || item.Seckey == nil || item.Pubkey == nil {
		return errors.New("input invalid addrItem")
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	bs, err := addrItemToByteSlice(item)
	if err != nil {
		return err
	}

	addrFilePath := filepath.Join(f.fileFolderPath, item.Addr)

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
	return nil
}

func (f *fileKeyStore) GetAddr(addr string) (addrItem, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	addrFilePath := filepath.Join(f.fileFolderPath, addr)
	exist, err := isPathExist(addrFilePath)
	if err != nil {
		return addrItem{}, err
	}
	if !exist {
		return addrItem{}, addrNotExistErr
	}

	data, err := ioutil.ReadFile(addrFilePath)
	if err != nil {
		return addrItem{}, err
	}

	var c crypt.CryptService
	switch f.CryptType {
	case tpcrtypes.CryptType_Secp256:
		c = new(secp256.CryptServiceSecp256)
	case tpcrtypes.CryptType_Ed25519:
		c = new(ed25519.CryptServiceEd25519)
	default:
		return addrItem{}, errors.New("unsupported CryptType")
	}
	decryptedAddr, err := c.StreamDecrypt(f.Seckey, data)
	if err != nil {
		return addrItem{}, err
	}

	return byteSliceToAddrItem(decryptedAddr)
}

func (f *fileKeyStore) List() (addrs []string, err error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	files, err := ioutil.ReadDir(f.fileFolderPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.Name() != walletEnableKey && file.Name() != defaultAddrKey {
			addrs = append(addrs, file.Name())
		}
	}
	return addrs, nil
}

func (f *fileKeyStore) SetWalletEnable(set bool) error {
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
	return nil
}

func (f *fileKeyStore) GetWalletEnable() (bool, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	weFile := filepath.Join(f.fileFolderPath, walletEnableKey)

	exist, err := isPathExist(weFile)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, walletEnableKeyNotSetErr
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
		return false, walletEnableKeyNotSetErr
	}
}

func (f *fileKeyStore) SetDefaultAddr(defaultAddr string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	daFilePath := filepath.Join(f.fileFolderPath, defaultAddrKey)

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

	var c crypt.CryptService
	switch f.CryptType {
	case tpcrtypes.CryptType_Secp256:
		c = new(secp256.CryptServiceSecp256)
	case tpcrtypes.CryptType_Ed25519:
		c = new(ed25519.CryptServiceEd25519)
	default:
		return errors.New("unsupported CryptType")
	}
	encryptedDA, err := c.StreamEncrypt(f.Pubkey, []byte(defaultAddr))
	if err != nil {
		return err
	}
	_, err = daFile.Write(encryptedDA)
	if err != nil {
		return err
	}
	return nil
}

func (f *fileKeyStore) GetDefaultAddr() (defaultAddr string, err error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	daFilePath := filepath.Join(f.fileFolderPath, defaultAddrKey)

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

		var c crypt.CryptService
		switch f.CryptType {
		case tpcrtypes.CryptType_Secp256:
			c = new(secp256.CryptServiceSecp256)
		case tpcrtypes.CryptType_Ed25519:
			c = new(ed25519.CryptServiceEd25519)
		default:
			return "", errors.New("unsupported CryptType")
		}
		decryptedDA, err := c.StreamDecrypt(f.Seckey, bs)
		if err != nil {
			return "", err
		}
		return string(decryptedDA), nil
	}
}

func (f *fileKeyStore) RemoveItem(key string) error {
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

	return os.Remove(fp)

}

func addrItemToByteSlice(item addrItem) (ret []byte, err error) {
	if item.Seckey == nil || item.Pubkey == nil || item.CryptType == tpcrtypes.CryptType_Unknown {
		return nil, errors.New("input invalid itemData")
	}

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err = enc.Encode(item)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func byteSliceToAddrItem(data []byte) (ret addrItem, err error) {
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
