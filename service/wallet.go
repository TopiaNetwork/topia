package service

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/wallet"
)

type WalletService interface {
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

func NewWalletService(w wallet.Wallet) WalletService {
	return &walletService{w}
}

type walletService struct {
	wallet.Wallet
}
