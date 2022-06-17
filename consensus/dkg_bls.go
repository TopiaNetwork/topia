package consensus

type DKGBls interface {
	PubKey() ([]byte, error)

	Sign(msg []byte) ([]byte, []byte, error) //Signature,PubKey

	Verify(msg, sig []byte) error

	VerifyAgg(msg, sig []byte) error

	RecoverSig(msg []byte, sigs [][]byte) ([]byte, error)

	Threshold() int

	Finished() bool
}

type DKGBLSUpdater interface {
	updateDKGBls(dkgBls DKGBls)
}
