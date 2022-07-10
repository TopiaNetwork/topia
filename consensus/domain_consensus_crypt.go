package consensus

import (
	"github.com/TopiaNetwork/kyber/v3"
	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/share"
	"github.com/TopiaNetwork/kyber/v3/sign/bls"
	"github.com/TopiaNetwork/kyber/v3/sign/tbls"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

type domainConsensusCrypt struct {
	threshold    int
	nParticipant int
	suite        *bn256.Suite
	publicKey    kyber.Point
	priShare     *share.PriShare
	pubShares    []*share.PubShare
}

func NewDomainConsensusCrypt(selfNode *tpcmm.NodeInfo, csDomain *tpcmm.NodeDomainInfo) (DKGBls, uint32) {
	suite := bn256.NewSuiteG2()
	pubKey := suite.Point()
	err := pubKey.UnmarshalBinary(csDomain.CSDomainData.PublicKey)
	if err != nil {
		panic("Public key UnmarshalBinary err: " + err.Error())
	}

	pubShares := make([]*share.PubShare, len(csDomain.CSDomainData.PubShares))
	for i, pubShareBytes := range csDomain.CSDomainData.PubShares {
		pubShares[i] = &share.PubShare{}
		pubShares[i].Unmarshal(pubShareBytes)
	}

	var priShare share.PriShare
	priShare.V = suite.Scalar()
	priShare.Unmarshal(selfNode.DKGPriShare)

	csDomainCrypt := &domainConsensusCrypt{
		threshold:    csDomain.CSDomainData.Threshold,
		nParticipant: csDomain.CSDomainData.NParticipant,
		suite:        suite,
		publicKey:    pubKey,
		priShare:     &priShare,
		pubShares:    pubShares,
	}

	if tpcmm.IsContainString(selfNode.NodeID, tpcmm.NodeIDs(csDomain.CSDomainData.Members)) {
		return csDomainCrypt, 1
	}

	return csDomainCrypt, 0
}

func (d *domainConsensusCrypt) PubKey() ([]byte, error) {
	return d.publicKey.MarshalBinary()
}

func (d *domainConsensusCrypt) PriShare() ([]byte, error) {
	return d.priShare.Marshal()
}

func (d *domainConsensusCrypt) PubShares() ([][]byte, error) {
	var pubSsBytes [][]byte

	for _, pubS := range d.pubShares {
		pubSBytes, err := pubS.Marshal()
		if err != nil {
			return nil, err
		}
		pubSsBytes = append(pubSsBytes, pubSBytes)
	}

	return pubSsBytes, nil
}

func (d *domainConsensusCrypt) Sign(msg []byte) ([]byte, []byte, error) {
	dInfo, err := tbls.Sign(d.suite, d.priShare, msg)
	if err != nil {
		return nil, nil, err
	}

	s := tbls.SigShare(dInfo)
	i, err := s.Index()
	if err != nil {
		return nil, nil, err
	}

	pubPoly, err := share.RecoverPubPoly(d.suite, d.pubShares, d.threshold, d.nParticipant)
	if err != nil {
		return nil, nil, err
	}

	pubSBytes, err := pubPoly.Eval(i).Marshal()

	return dInfo, pubSBytes, err
}

func (d *domainConsensusCrypt) Verify(msg, sig []byte) error {
	pubPoly, err := share.RecoverPubPoly(d.suite, d.pubShares, d.threshold, d.nParticipant)
	if err != nil {
		return err
	}

	s := tbls.SigShare(sig)
	i, err := s.Index()
	if err != nil {
		return err
	}

	return bls.Verify(d.suite, pubPoly.Eval(i).V, msg, s.Value())
}

func (d *domainConsensusCrypt) VerifyAgg(msg, sig []byte) error {
	return bls.Verify(d.suite, d.publicKey, msg, sig)
}

func (d *domainConsensusCrypt) RecoverSig(msg []byte, sigs [][]byte) ([]byte, error) {
	pubPoly, err := share.RecoverPubPoly(d.suite, d.pubShares, d.threshold, d.nParticipant)
	if err != nil {
		return nil, err
	}

	return tbls.Recover(d.suite, pubPoly, msg, sigs, d.threshold, d.nParticipant)
}

func (d *domainConsensusCrypt) Threshold() int {
	return d.threshold
}

func (d *domainConsensusCrypt) Finished() bool {
	return true
}
