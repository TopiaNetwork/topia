package consensus

type DKGBls interface {
	Sign(msg []byte) ([]byte, []byte, error) //Signature,PubKey

	Verify(msg, sig []byte) error

	RecoverSig(msg []byte, sigs [][]byte) ([]byte, error)
}

type DKGBLSUpdater interface {
	updateDKGBls(dkgBls DKGBls)
}