package types

type ProtocolID byte

type PrivateKey []byte

type PublicKey []byte

type Signature []byte

type CryptType byte

const (
	CryptType_Unknown CryptType = iota
	CryptType_BLS12381
	CryptType_Secp256
)
