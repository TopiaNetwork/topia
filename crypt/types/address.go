package types

import (
	"bytes"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpnet "github.com/TopiaNetwork/topia/network"
)

const (
	AddressLen_ETH      = 20 //20 bytes
	AddressLen_Secp256  = 20 //20 bytes
	AddressLen_BLS12381 = 48 //48 bytes
	AddressLen_ED25519  = 20 //20 bytes
)

const (
	MainnetPrefix = "m"
	TestnetPrefix = "t"
)

const checksumHashLength = 4

const encodeStd = "topianewrkbcdfghjlmqsuvxyz123456"

var addressEncoding = base32.NewEncoding(encodeStd)

var UndefAddress = Address("<empty>")

type Address string

func checksum(data []byte) []byte {
	return tpcmm.NewBlake2bHasher(checksumHashLength).Compute(string(data))
}

func validateChecksum(data, expect []byte) bool {
	digest := checksum(data)
	return bytes.Equal(digest, expect)
}

func encode(network tpnet.NetworkType, cryptType CryptType, payload []byte) (Address, error) {
	if len(payload) == 0 {
		return UndefAddress, errors.New("Invalid payload: len 0")
	}
	var ntwPrefix string
	switch network {
	case tpnet.NetworkType_Mainnet:
		ntwPrefix = MainnetPrefix
	case tpnet.NetworkType_Testnet:
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

func decode(a string) (tpnet.NetworkType, CryptType, []byte, error) {
	if len(a) == 0 {
		return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, nil
	}
	if a == string(UndefAddress) {
		return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, nil
	}

	if string(a[0]) != MainnetPrefix && string(a[0]) != TestnetPrefix {
		return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Unknown network type %d", a[0])
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
		return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Unknown crypt type %d", cryptType)
	}

	raw := a[2:]

	payloadcksm, err := addressEncoding.WithPadding(-1).DecodeString(raw)
	if err != nil {
		return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, err
	}

	if len(payloadcksm)-checksumHashLength < 0 {
		return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid checksum %d", len(payloadcksm))
	}

	payload := payloadcksm[:len(payloadcksm)-checksumHashLength]
	cksm := payloadcksm[len(payloadcksm)-checksumHashLength:]

	if cryptType == CryptType_Secp256 {
		if len(payload) != AddressLen_Secp256 {
			return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid payload %d, expected %d", len(payload), AddressLen_Secp256)
		}
	} else if cryptType == CryptType_BLS12381 {
		if len(payload) != AddressLen_BLS12381 {
			return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid payload %d, expected %d", len(payload), AddressLen_BLS12381)
		}
	} else if cryptType == CryptType_Ed25519 {
		if len(payload) != AddressLen_ED25519 {
			return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid payload %d, expected=%d", len(payload), AddressLen_ED25519)
		}
	}

	if !validateChecksum(append([]byte{byte(cryptType)}, payload...), cksm) {
		return tpnet.NetworkType_Unknown, CryptType_Unknown, nil, fmt.Errorf("Invalid checksum")
	}

	return tpnet.NetworkType(0).Value(a[0]), cryptType, payload, nil
}

func NewAddress(cryptType CryptType, payload []byte) (Address, error) {
	return encode(tpnet.CurrentNetworkType, cryptType, payload)
}

func NewFromString(addr string) Address {
	return Address(addr)
}

func NewFromBytes(addr []byte) Address {
	return Address(addr)
}

func (a Address) NetworkType() (tpnet.NetworkType, error) {
	if len(a) == 0 {
		return tpnet.NetworkType_Unknown, errors.New("Invalid address: len 0")
	}

	netType, _, _, err := decode(string(a))
	if err != nil {
		netType = tpnet.NetworkType_Unknown
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

func (a Address) IsValid(netType tpnet.NetworkType, cryptType CryptType) (bool, error) {
	nType, cType, _, err := decode(string(a))
	if err != nil {
		return false, err
	}

	if nType != netType {
		return false, fmt.Errorf("Invalid net type: expected %s, actual %s", netType.String(), nType.String())
	}
	if cType == cryptType {
		return false, fmt.Errorf("Invalid crypt type: expected %d, actual %s", cryptType.String(), cType.String())
	}

	return true, nil
}

func (a Address) IsHexs() bool {
	t := string(a)
	if tpcmm.Has0xPrefix(string(a)) {
		t = string(a[2:])
	}

	hexSize := len(t)

	return (hexSize == 2*AddressLen_ETH || hexSize == 2*AddressLen_Secp256 || hexSize == AddressLen_BLS12381) && tpcmm.IsHex(t)
}

func (a Address) IsEth() bool {
	t := string(a)
	if tpcmm.Has0xPrefix(string(a)) {
		t = string(a[2:])
	}

	if (len(t) == 2*AddressLen_ETH && tpcmm.IsHex(t)) || (len(t) == AddressLen_ETH) {
		return true
	}

	return false
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
