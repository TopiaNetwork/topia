package types

import (
	"bytes"
	"encoding/binary"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

var NativeContractAddr_Account Address

func init() {
	NativeContractAddr_Account = CreateNativeContractAddress(1)
}

func CreateNativeContractAddress(id uint64) Address {
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, id)

	var ntwPrefix string
	switch tpcmm.CurrentNetworkType {
	case tpcmm.NetworkType_Mainnet:
		ntwPrefix = MainnetPrefix
	case tpcmm.NetworkType_Testnet:
		ntwPrefix = TestnetPrefix
	default:
		return UndefAddress
	}

	addrDataPrefix := bytes.Repeat([]byte{0x00}, 12)
	addrData := append(addrDataPrefix, idBytes...)
	strAddr := ntwPrefix + "0" + addressEncoding.WithPadding(-1).EncodeToString(addrData)

	return Address(strAddr)
}

func CreateUserContractAddress(fromAddr Address, nonce uint64) Address {
	if fromAddr == "" || fromAddr == UndefAddress {
		panic("Invalid address" + string(fromAddr))
	}

	fromPayload, _ := fromAddr.Payload()
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)

	hasher := tpcmm.NewBlake2bHasher(AddressLen_ED25519)
	hasher.Writer().Write(fromPayload[:]) // does not error
	hasher.Writer().Write(nonceBytes)
	newAddr, err := NewAddress(CryptType_Ed25519, hasher.Bytes())

	if err != nil {
		newAddr = UndefAddress
	}

	return newAddr
}
