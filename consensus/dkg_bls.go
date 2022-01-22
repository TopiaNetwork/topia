package consensus

type DKGBls interface {
	Sign(msg []byte) ([]byte, error)

	Verify(msg, sig []byte) error

	RecoverSig(msg []byte, sigs [][]byte) ([]byte, error)
}
