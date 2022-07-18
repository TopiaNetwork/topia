package wallet

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/crypt"
	"github.com/TopiaNetwork/topia/crypt/ed25519"
	"github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
	"io/ioutil"
)

const (
	MOD_NAME = "wallet"
)

type Wallet interface {
	Create(cryptType tpcrtypes.CryptType) (tpcrtypes.Address, error)

	CreateMnemonic(cryptType tpcrtypes.CryptType, passphrase string, mnemonicAmounts int) (mnemonic string, err error)

	Recovery(cryptType tpcrtypes.CryptType, mnemonic string, passphrase string) (tpcrtypes.Address, error)

	Import(cryptType tpcrtypes.CryptType, privKey tpcrtypes.PrivateKey) (tpcrtypes.Address, error)

	Delete(tpcrtypes.Address) error

	SetDefault(tpcrtypes.Address) error

	Default() (tpcrtypes.Address, error)

	Export(tpcrtypes.Address) (tpcrtypes.PrivateKey, error)

	Sign(addr tpcrtypes.Address, msg []byte) (tpcrtypes.SignatureInfo, error)

	Has(tpcrtypes.Address) (bool, error)

	List() ([]tpcrtypes.Address, error)

	Lock(addr tpcrtypes.Address, lock bool) error

	IsLocked(addr tpcrtypes.Address) (bool, error)

	Enable(set bool) error

	IsEnable() (bool, error)
}

// EncryptWayOfWallet hold keys' file encryption and decryption arguments when using "topiaKeyStore" as walletBackend.
type EncryptWayOfWallet struct {
	CryptType tpcrtypes.CryptType
	Pubkey    tpcrtypes.PublicKey
	Seckey    tpcrtypes.PrivateKey
}

type wallet struct {
	log      tplog.Logger
	rootPath string
	ws       walletStore
}

type userConfig struct {
	WalletBackend  string `json:"walletBackend"`
	KeyringBackend string `json:"keyringBackend"`
}

var walletBackendConfig userConfig

func init() { // load wallet config json
	var log tplog.Logger
	data, err := ioutil.ReadFile("./wallet_config.json")
	if err != nil {
		log.Fatal("missing wallet config file")
		panic(err)
	}
	err = json.Unmarshal(data, &walletBackendConfig)
	if err != nil {
		log.Fatal("wallet config file might be corrupted")
		panic(err)
	}
}

func NewWallet(level tplogcmm.LogLevel, log tplog.Logger, rootPath string, encryptWay EncryptWayOfWallet) (Wallet, error) {
	wLog := tplog.CreateModuleLogger(level, MOD_NAME, log)

	wsImp, err := loadWalletConfig(rootPath, encryptWay)
	if err != nil {
		return nil, err
	}
	var w = wallet{
		log:      wLog,
		rootPath: rootPath,
		ws:       wsImp,
	}

	return &w, nil
}

func (w *wallet) Create(cryptType tpcrtypes.CryptType) (tpcrtypes.Address, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return "", err
	}
	if walletEnable != true {
		return "", errors.New("wallet is not enabled")
	}

	var c crypt.CryptService
	if cryptType == tpcrtypes.CryptType_Secp256 {
		c = new(secp256.CryptServiceSecp256)
	} else if cryptType == tpcrtypes.CryptType_Ed25519 {
		c = new(ed25519.CryptServiceEd25519)
	} else {
		return "", errors.New("your input cryptType is not supported")
	}

	sec, pub, err := c.GeneratePriPubKey()
	if err != nil {
		return "", err
	}
	addr, err := c.CreateAddress(pub)
	if err != nil {
		return "", err
	}
	err = w.ws.SetAddr(addrItem{
		Addr:       string(addr),
		AddrLocked: false,
		Seckey:     sec,
		Pubkey:     pub,
		CryptType:  c.CryptType(),
	})
	if err != nil {
		return "", err
	}
	return addr, nil
}

func (w *wallet) CreateMnemonic(cryptType tpcrtypes.CryptType, passphrase string, mnemonicAmounts int) (string, error) {
	if mnemonicAmounts != 12 && mnemonicAmounts != 24 {
		return "", errors.New("don't support input mnemonic amounts other than 12 and 24")
	}

	randomNum := make([]byte, mnemonicAmounts/12*16)
	_, err := rand.Read(randomNum)
	if err != nil {
		return "", err
	}

	mnemonic, err := bip39.NewMnemonic(randomNum)
	if err != nil {
		return "", err
	}

	seed := pbkdf2.Key([]byte(mnemonic), []byte("mnemonic"+passphrase), 4096, 32, sha256.New)

	var c crypt.CryptService
	if cryptType == tpcrtypes.CryptType_Secp256 {
		c = new(secp256.CryptServiceSecp256)
	} else if cryptType == tpcrtypes.CryptType_Ed25519 {
		c = new(ed25519.CryptServiceEd25519)
	} else {
		return "", errors.New("unsupported cryptType:" + cryptType.String())
	}

	sec, pub, err := c.GeneratePriPubKeyBySeed(seed)
	if err != nil {
		return "", err
	}
	addr, err := c.CreateAddress(pub)
	if err != nil {
		return "", err
	}
	item := addrItem{
		Addr:       string(addr),
		AddrLocked: false,
		Seckey:     sec,
		Pubkey:     pub,
		CryptType:  cryptType,
	}
	err = w.ws.SetAddr(item)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func (w *wallet) Recovery(cryptType tpcrtypes.CryptType, mnemonic string, passphrase string) (tpcrtypes.Address, error) {
	seed := pbkdf2.Key([]byte(mnemonic), []byte("mnemonic"+passphrase), 4096, 32, sha256.New)

	var c crypt.CryptService
	if cryptType == tpcrtypes.CryptType_Secp256 {
		c = new(secp256.CryptServiceSecp256)
	} else if cryptType == tpcrtypes.CryptType_Ed25519 {
		c = new(ed25519.CryptServiceEd25519)
	} else {
		return "", errors.New("unsupported cryptType:" + cryptType.String())
	}

	sec, pub, err := c.GeneratePriPubKeyBySeed(seed)
	if err != nil {
		return "", err
	}
	addr, err := c.CreateAddress(pub)
	if err != nil {
		return "", err
	}

	_, err = w.ws.GetAddr(string(addr))
	if err != nil {
		item := addrItem{
			Addr:       string(addr),
			AddrLocked: false,
			Seckey:     sec,
			Pubkey:     pub,
			CryptType:  cryptType,
		}
		err = w.ws.SetAddr(item)
		if err != nil {
			return "", err
		}
		return addr, nil
	}
	return addr, nil
}

func (w *wallet) Import(cryptType tpcrtypes.CryptType, privKey tpcrtypes.PrivateKey) (tpcrtypes.Address, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return "", err
	}
	if walletEnable != true {
		return "", errors.New("wallet is not enabled")
	}

	var c crypt.CryptService

	if cryptType == tpcrtypes.CryptType_Secp256 {
		c = new(secp256.CryptServiceSecp256)
	} else if cryptType == tpcrtypes.CryptType_Ed25519 {
		c = new(ed25519.CryptServiceEd25519)
	} else {
		return "", errors.New("your input cryptType is not supported")
	}

	pub, err := c.ConvertToPublic(privKey)
	if err != nil {
		return "", err
	}
	addr, err := c.CreateAddress(pub)
	if err != nil {
		return "", err
	}
	err = w.ws.SetAddr(addrItem{
		Addr:       string(addr),
		AddrLocked: false,
		Seckey:     privKey,
		Pubkey:     pub,
		CryptType:  c.CryptType(),
	})
	if err != nil {
		return "", err
	}
	return addr, nil
}

func (w *wallet) Delete(address tpcrtypes.Address) error {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return err
	}
	if walletEnable != true {
		return errors.New("wallet is not enabled")
	}

	item, err := w.ws.GetAddr(string(address))
	if err != nil {
		return err
	}
	if item.AddrLocked {
		return errors.New("this addr has been locked")
	}

	err = w.ws.RemoveItem(string(address))
	if err != nil {
		return err
	}
	return nil
}

func (w *wallet) SetDefault(address tpcrtypes.Address) error {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return err
	}
	if walletEnable != true {
		return errors.New("wallet is not enabled")
	}

	item, err := w.ws.GetAddr(string(address))
	if err != nil {
		return err
	}
	if item.AddrLocked {
		return errors.New("this addr has been locked")
	}

	err = w.ws.SetDefaultAddr(string(address))
	if err != nil {
		return err
	}
	return nil
}

func (w *wallet) Default() (tpcrtypes.Address, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return "", err
	}
	if walletEnable != true {
		return "", errors.New("wallet is not enabled")
	}

	addr, err := w.ws.GetDefaultAddr()
	if err != nil {
		return "", err
	}
	return tpcrtypes.Address(addr), nil
}

func (w *wallet) Export(address tpcrtypes.Address) (tpcrtypes.PrivateKey, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return nil, err
	}
	if walletEnable != true {
		return nil, errors.New("wallet is not enabled")
	}

	item, err := w.ws.GetAddr(string(address))
	if err != nil {
		return nil, err
	}
	if item.AddrLocked {
		return nil, errors.New("this addr has been locked")
	}

	return item.Seckey, nil
}

func (w *wallet) Sign(addr tpcrtypes.Address, msg []byte) (tpcrtypes.SignatureInfo, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return tpcrtypes.SignatureInfo{}, err
	}
	if walletEnable != true {
		return tpcrtypes.SignatureInfo{}, errors.New("wallet is not enabled")
	}

	item, err := w.ws.GetAddr(string(addr))
	if err != nil {
		return tpcrtypes.SignatureInfo{}, err
	}
	if item.AddrLocked {
		return tpcrtypes.SignatureInfo{}, errors.New("this addr has been locked")
	}

	var c crypt.CryptService
	if item.CryptType == tpcrtypes.CryptType_Secp256 {
		c = new(secp256.CryptServiceSecp256)
	} else if item.CryptType == tpcrtypes.CryptType_Ed25519 {
		c = new(ed25519.CryptServiceEd25519)
	} else {
		return tpcrtypes.SignatureInfo{}, errors.New("unsupported cryptType")
	}

	sig, err := c.Sign(item.Seckey, msg)
	if err != nil {
		return tpcrtypes.SignatureInfo{}, err
	}
	return tpcrtypes.SignatureInfo{SignData: sig, PublicKey: item.Pubkey}, nil
}

func (w *wallet) Has(address tpcrtypes.Address) (bool, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return false, err
	}
	if walletEnable != true {
		return false, errors.New("wallet is not enabled")
	}

	_, err = w.ws.GetAddr(string(address))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (w *wallet) List() ([]tpcrtypes.Address, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return nil, err
	}
	if walletEnable != true {
		return nil, errors.New("wallet is not enabled")
	}

	addrs, err := w.ws.List()
	if err != nil {
		return nil, err
	}
	ret := make([]tpcrtypes.Address, len(addrs))
	for i := range addrs {
		ret[i] = tpcrtypes.Address(addrs[i])
	}
	return ret, nil
}

func (w *wallet) Lock(addr tpcrtypes.Address, lock bool) error {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return err
	}
	if walletEnable != true {
		return errors.New("wallet is not enabled")
	}

	item, err := w.ws.GetAddr(string(addr))
	if err != nil {
		return err
	}

	if item.AddrLocked == lock {
		return nil
	}

	item.AddrLocked = lock
	err = w.ws.SetAddr(item)
	if err != nil {
		return err
	}
	return nil
}

func (w *wallet) IsLocked(addr tpcrtypes.Address) (bool, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return true, err
	}
	if walletEnable != true {
		return true, errors.New("wallet is not enabled")
	}

	item, err := w.ws.GetAddr(string(addr))
	if err != nil {
		return true, err
	}
	return item.AddrLocked, nil
}

func (w *wallet) Enable(set bool) error {
	return w.ws.SetWalletEnable(set)
}

func (w *wallet) IsEnable() (bool, error) {
	walletEnable, err := w.ws.GetWalletEnable()
	if err != nil {
		return false, err
	}
	return walletEnable, nil
}

func loadWalletConfig(rootPath string, encryptWay EncryptWayOfWallet) (wsImp walletStore, err error) {
	var initArgument interface{}
	if walletBackendConfig.WalletBackend == "topiaKeyStore" {
		var c crypt.CryptService
		switch encryptWay.CryptType {
		case tpcrtypes.CryptType_Secp256:
			c = new(secp256.CryptServiceSecp256)
		case tpcrtypes.CryptType_Ed25519:
			c = new(ed25519.CryptServiceEd25519)
		default:
			return nil, errors.New("unsupported CryptType")
		}

		pubConvert, err := c.ConvertToPublic(encryptWay.Seckey)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(pubConvert, encryptWay.Pubkey) {
			return nil, errors.New("input keypair is not valid")
		}

		wsImp = new(fileKeyStore)
		initArgument = initArg{
			RootPath:           rootPath,
			EncryptWayOfWallet: encryptWay,
		}
	} else if walletBackendConfig.WalletBackend == "keyring" {
		wsImp = new(keyringImp)
		initArgument = keyringInitArg{
			RootPath: rootPath,
			Backend:  walletBackendConfig.KeyringBackend,
		}
	} else {
		return nil, errors.New("unknown wallet backend, please check wallet config file")
	}

	if err = wsImp.Init(initArgument); err != nil {
		return nil, err
	}
	return wsImp, nil
}
