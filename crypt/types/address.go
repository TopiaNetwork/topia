package types

import (
	"bytes"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

const (
	AddressLen_ETH      = 20 //20 bytes
	AddressLen_Secp256  = 20 //20 bytes
	AddressLen_BLS12381 = 48 //48 bytes
	AddressLen_ED25519  = 32 //32 bytes
)

const (
	MainnetPrefix = "m"
	TestnetPrefix = "t"
)

const checksumHashLength = 4

const encodeStd = "topianewrkbcdfghjlmqsuvxyz123456"

var addressEncoding = base32.NewEncoding(encodeStd)

var UndefAddress = Address("<empty>")

// A Full address contains: netType(1 byte) + cryptType(1 byte) + payload(different size as different cryptType + checkSum(4 byte)
type Address string

func checksum(data []byte) []byte {
	return tpcmm.NewBlake2bHasher(checksumHashLength).Compute(string(data))
}

func validateChecksum(data, expect []byte) bool {
	digest := checksum(data)
	return bytes.Equal(digest, expect)
}

func encode(network tpcmm.NetworkType, cryptType CryptType, payload []byte) (Address, error) {
	if len(payload) == 0 {
		return UndefAddress, errors.New("Invalid payload: len 0")
	}
	var ntwPrefix string
	switch network {
	case tpcmm.NetworkType_Mainnet:
		ntwPrefix = MainnetPrefix
	case tpcmm.NetworkType_Testnet:
		ntwPrefix = TestnetPrefix
	default:
		return UndefAddress, fmt.Errorf("Unknown network type %d", network)
	}

	var strAddr string
	switch cryptType {
	case CryptType_BLS12381, CryptType_Secp256, CryptType_Ed25519:
		cksm := checksum(append([]byte{byte(cryptType)}, payload...))
		strAddr = ntwPrefix + fmt.Sprintf("%d", cryptType) + addressEncoding.WithPadding(-1).EncodeToString(append(payload, cksm[:]...))
	default:
		return UndefAddress, fmt.Errorf("Unknown crypt type %d", cryptType)
	}

	return Address(strAddr), nil
}

func decode(a string) (tpcmm.NetworkType, CryptType, []byte, error) {
	if len(a) == 0 {
		return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, nil
	}
	if a == string(UndefAddress) {
		return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, nil
	}

	if string(a[0]) != MainnetPrefix && string(a[0]) != TestnetPrefix {
		return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Unknown network type %d", a[0])
	}

	var cryptType CryptType
	switch a[1] {
	case '1':
		cryptType = CryptType_BLS12381
	case '2':
		cryptType = CryptType_Secp256
	case '3':
		cryptType = CryptType_Ed25519
	default:
		return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Unknown crypt type %d", cryptType)
	}

	raw := a[2:]

	payloadcksm, err := addressEncoding.WithPadding(-1).DecodeString(raw)
	if err != nil {
		return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, err
	}

	if len(payloadcksm)-checksumHashLength < 0 {
		return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid checksum %d", len(payloadcksm))
	}

	payload := payloadcksm[:len(payloadcksm)-checksumHashLength]
	cksm := payloadcksm[len(payloadcksm)-checksumHashLength:]

	if cryptType == CryptType_Secp256 {
		if len(payload) != AddressLen_Secp256 {
			return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid payload %d, expected %d", len(payload), AddressLen_Secp256)
		}
	} else if cryptType == CryptType_BLS12381 {
		if len(payload) != AddressLen_BLS12381 {
			return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid payload %d, expected %d", len(payload), AddressLen_BLS12381)
		}
	} else if cryptType == CryptType_Ed25519 {
		if len(payload) != AddressLen_ED25519 {
			return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid payload %d, expected=%d", len(payload), AddressLen_ED25519)
		}
	}

	if !validateChecksum(append([]byte{byte(cryptType)}, payload...), cksm) {
		return tpcmm.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid checksum")
	}

	return tpcmm.NetworkType(0).Value(a[0]), cryptType, payload, nil
}

func NewAddress(cryptType CryptType, payload []byte) (Address, error) {
	return encode(tpcmm.CurrentNetworkType, cryptType, payload)
}

func NewFromString(addr string) Address {
	_, _, _, err := decode(addr)
	if err != nil {
		return UndefAddress
	}

	return Address(addr)
}

func NewFromBytes(addr []byte) Address {
	return Address(addr)
}

func (a Address) NetworkType() (tpcmm.NetworkType, error) {
	if len(a) == 0 {
		return tpcmm.NetworkType_Unknown, errors.New("Invalid address: len 0")
	}

	netType, _, _, err := decode(string(a))
	if err != nil {
		netType = tpcmm.NetworkType_Unknown
	}

	return netType, err
}

func (a Address) CryptType() (CryptType, error) {
	if len(a) == 0 {
		return CryptType_Unknown, errors.New("Invalid address: len 0")
	}

	_, cryptType, _, err := decode(string(a))
	if err != nil {
		cryptType = CryptType_Unknown
	}

	return cryptType, err
}

func (a Address) HexString() (string, error) {
	if !a.IsHexs() {
		return fmt.Sprintf("%d%d%s", a[0], a[1], hex.EncodeToString([]byte(a[2:]))), nil
	}

	return string(a), nil
}

func (a Address) IsValid(netType tpcmm.NetworkType) bool {
	nType, _, _, err := decode(string(a))
	if err != nil {
		return false
	}

	if nType != netType {
		return false
	}

	return true
}

func (a Address) IsHexs() bool {
	t := string(a)
	if tpcmm.Has0xPrefix(string(a)) {
		t = string(a[2:])
	}

	hexSize := len(t)

	return (hexSize == 2*AddressLen_ETH || hexSize == 2*AddressLen_Secp256 || hexSize == AddressLen_BLS12381) && tpcmm.IsHex(t)
}

func (a Address) Payload() ([]byte, error) {
	if len(a) == 0 {
		return nil, errors.New("Invalid address: len 0")
	}

	_, _, pLoad, err := decode(string(a))
	if err != nil {
		pLoad = nil
	}

	return pLoad, err
}

func (a Address) Bytes() []byte {
	return []byte(a)
}

func (a Address) IsPayable() bool {
	return a != NativeContractAddr_Account
}

func IsEth(a string) bool {
	t := a
	if tpcmm.Has0xPrefix(a) {
		t = a[2:]
	}

	if (len(t) == 2*AddressLen_ETH && tpcmm.IsHex(t)) || (len(t) == AddressLen_ETH) {
		return true
	}

	return false
}

func TopiaAddressFromEth(a string) (Address, error) {
	if len(a) < AddressLen_ETH {
		return UndefAddress, fmt.Errorf("Invalid eth address len: %s", a)
	}

	t := a
	tBytes := []byte(a)
	if tpcmm.Has0xPrefix(a) {
		t = a[2:]
		if len(t)%2 == 1 {
			t = "0" + t
		}
		tBytes, _ = hex.DecodeString(t)
	}

	return NewAddress(CryptType_Secp256, tBytes)
}

func EthAddressFromTopia(a Address) ([]byte, error) {
	_, cType, pLoad, err := decode(string(a))
	if err != nil {
		return nil, err
	}

	if cType != CryptType_Secp256 {
		return nil, fmt.Errorf("Invalid crypt address type: %s", cType.String())
	}

	return pLoad, nil
}
