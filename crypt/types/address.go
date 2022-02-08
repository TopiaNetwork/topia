package types

import (
	"bytes"
	"encoding/base32"
	"errors"
	"fmt"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

type NetworkType = byte

const (
	NetworkType_Unknown NetworkType = iota
	NetworkType_Mainnet
	NetworkType_Testnet
)

const (
	MainnetPrefix = "m"
	TestnetPrefix = "t"
)

const checksumHashLength = 4

const encodeStd = "topianetworkqscvghtyeoplkmwcxzah"

var addressEncoding = base32.NewEncoding(encodeStd)

var UndefAddress = Address("<empty>")

var CurrentNetworkType = NetworkType_Testnet

type Address string

func checksum(data []byte) []byte {
	return tpcmm.NewBlake2bHasher(checksumHashLength).Compute(string(data))
}

func validateChecksum(data, expect []byte) bool {
	digest := checksum(data)
	return bytes.Equal(digest, expect)
}

func encode(network NetworkType, cryptType CryptType, payload []byte) (Address, error) {
	if len(payload) == 0 {
		return UndefAddress, errors.New("Invalid payload: len 0")
	}
	var ntwPrefix string
	switch network {
	case NetworkType_Mainnet:
		ntwPrefix = MainnetPrefix
	case NetworkType_Testnet:
		ntwPrefix = TestnetPrefix
	default:
		return UndefAddress, fmt.Errorf("Unknown network type %d", network)
	}

	var strAddr string
	switch cryptType {
	case CryptType_BLS12381, CryptType_Secp256:
		cksm := checksum(append([]byte{byte(cryptType)}, payload...))
		strAddr = ntwPrefix + fmt.Sprintf("%d", cryptType) + addressEncoding.WithPadding(-1).EncodeToString(append(payload, cksm[:]...))
	default:
		return UndefAddress, fmt.Errorf("Unknown crypt type %d", cryptType)
	}

	return Address(strAddr), nil
}

func decode(a string) (NetworkType, CryptType, []byte, error) {
	if len(a) == 0 {
		return NetworkType_Unknown, CryptType_Unknown, nil, nil
	}
	if a == string(UndefAddress) {
		return NetworkType_Unknown, CryptType_Unknown, nil, nil
	}

	if string(a[0]) != MainnetPrefix && string(a[0]) != TestnetPrefix {
		return NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Unknown network type %d", a[0])
	}

	var cryptType CryptType
	switch a[1] {
	case '1':
		cryptType = CryptType_BLS12381
	case '2':
		cryptType = CryptType_Secp256
	default:
		return NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Unknown crypt type %d", cryptType)
	}

	raw := a[2:]

	payloadcksm, err := addressEncoding.WithPadding(-1).DecodeString(raw)
	if err != nil {
		return NetworkType_Unknown, CryptType_Unknown, nil, err
	}

	if len(payloadcksm)-checksumHashLength < 0 {
		return NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid checksum %d", len(payloadcksm))
	}

	payload := payloadcksm[:len(payloadcksm)-checksumHashLength]
	cksm := payloadcksm[len(payloadcksm)-checksumHashLength:]

	if cryptType == CryptType_Secp256 {
		if len(payload) != 20 {
			return NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid payload %d", len(payload))
		}
	}

	if !validateChecksum(append([]byte{byte(cryptType)}, payload...), cksm) {
		return NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid checksum")
	}

	return a[0], cryptType, payload, nil
}

func NewAddress(cryptType CryptType, payload []byte) (Address, error) {
	return encode(CurrentNetworkType, cryptType, payload)
}

func NewFromString(addr string) Address {
	return Address(addr)
}

func NewFromBytes(addr []byte) Address {
	return Address(addr)
}

func (a Address) NetworkType() (NetworkType, error) {
	if len(a) == 0 {
		return NetworkType_Unknown, errors.New("Invalid address: len 0")
	}

	netType, _, _, err := decode(string(a))
	if err != nil {
		netType = NetworkType_Unknown
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
