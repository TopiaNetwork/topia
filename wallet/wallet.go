package wallet

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

const (
	MOD_NAME = "wallet"
)

type Wallet interface {
	Create(cryptType tpcrtypes.CryptType) (tpcrtypes.Address, error)

	Recovery(cryptType tpcrtypes.CryptType, mnemonic string, passphrase string) (tpcrtypes.Address, error)

	Import(privKey tpcrtypes.PrivateKey) (tpcrtypes.Address, error)

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

type wallet struct {
	log      tplog.Logger
	rootPath string
}

func NewWallet(level tplogcmm.LogLevel, log tplog.Logger, rootPath string) Wallet {
	wLog := tplog.CreateModuleLogger(level, MOD_NAME, log)
	return &wallet{wLog, rootPath}
}

func (w *wallet) Create(cryptType tpcrtypes.CryptType) (tpcrtypes.Address, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Recovery(cryptType tpcrtypes.CryptType, mnemonic string, passphrase string) (tpcrtypes.Address, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Import(privKey tpcrtypes.PrivateKey) (tpcrtypes.Address, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Delete(address tpcrtypes.Address) error {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) SetDefault(address tpcrtypes.Address) error {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Default() (tpcrtypes.Address, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Export(address tpcrtypes.Address) (tpcrtypes.PrivateKey, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Sign(addr tpcrtypes.Address, msg []byte) (tpcrtypes.SignatureInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Has(address tpcrtypes.Address) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) List() ([]tpcrtypes.Address, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Lock(addr tpcrtypes.Address, lock bool) error {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) IsLocked(addr tpcrtypes.Address) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) Enable(set bool) error {
	//TODO implement me
	panic("implement me")
}

func (w *wallet) IsEnable() (bool, error) {
	//TODO implement me
	panic("implement me")
}

