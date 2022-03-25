package types

type ProtocolID byte

type PrivateKey []byte

type PublicKey []byte

type Signature []byte

type SignatureInfo struct {
	SignData  Signature
	PublicKey []byte
}

type CryptType byte

const (
	CryptType_Unknown CryptType = iota
	CryptType_BLS12381
	CryptType_Secp256
	CryptType_Ed25519
)

func (c CryptType) String() string {
	switch c {
	case CryptType_BLS12381:
		return "BLS12381"
	case CryptType_Secp256:
		return "Secp256"
	case CryptType_Ed25519:
		return "Ed25519"
	default:
		return "Unknown"
	}
}

func (c CryptType) Value(crypyType string) CryptType {
	switch crypyType {
	case "BLS12381":
		return CryptType_BLS12381
	case "Secp256":
		return CryptType_Secp256
	case "Ed25519":
		return CryptType_Ed25519
	default:
		return CryptType_Unknown
	}
}
