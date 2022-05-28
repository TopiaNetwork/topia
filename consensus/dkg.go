package consensus

import (
	"bytes"
	"errors"
	"fmt"
	vss "github.com/TopiaNetwork/kyber/v3/share/vss/pedersen"
	"sync"

	"github.com/TopiaNetwork/kyber/v3"
	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/share"
	dkg "github.com/TopiaNetwork/kyber/v3/share/dkg/pedersen"
	"github.com/TopiaNetwork/kyber/v3/sign/bls"
	"github.com/TopiaNetwork/kyber/v3/sign/tbls"
	"github.com/TopiaNetwork/kyber/v3/util/encoding"

	tplog "github.com/TopiaNetwork/topia/log"
)

type TestCacheData struct {
	pubP kyber.Point
	msg  []byte
	sign []byte
}

type dkgCrypt struct {
	log tplog.Logger
	//index           uint32
	epoch           uint64
	initPrivKey     kyber.Scalar
	partPubKeysSync sync.RWMutex
	initPartPubKeys []kyber.Point
	threshold       int
	nParticipant    int
	suite           *bn256.Suite
	remoteDealsSync sync.RWMutex
	remoteDeals     []*dkg.Deal
	remoteRespsSync sync.RWMutex
	remoteAdvanResp []*dkg.Response
	dkGenerator     *dkg.DistKeyGenerator
}

func newDKGCrypt(log tplog.Logger /*index uint32, */, epoch uint64 /*suite *bn256.Suite, */, initPrivKey string, initPartPubKeys []string, threshold int, nParticipant int) *dkgCrypt {
	if len(initPrivKey) == 0 {
		log.Panicf("Blank initPrivKey %s", initPrivKey)
	}

	if len(initPartPubKeys) == 0 {
		log.Panic("Blank initPartPubKeys")
	}

	log.Infof("When newDKGCrypt: initPartPubKeys=%v", initPartPubKeys)

	suite := bn256.NewSuiteG2()
	priKey, err := encoding.StringHexToScalar(suite, initPrivKey)
	if err != nil {
		log.Panicf("Invalid initPrivKey %s", initPrivKey)
	}

	if len(initPartPubKeys) != nParticipant {
		log.Panicf("Invalid initPartPubKeys number, expected %d actual ", nParticipant, len(initPartPubKeys))
	}

	initPartPubKeyP := make([]kyber.Point, nParticipant)
	for i := 0; i < len(initPartPubKeys); i++ {
		partPubKey, err := encoding.StringHexToPoint(suite, initPartPubKeys[i])
		if err != nil {
			log.Panicf("Invalid initPubKey %d %s", i, initPartPubKeys[i])
		}
		initPartPubKeyP[i] = partPubKey
	}

	dkgCrypt := &dkgCrypt{
		log: log,
		//index:           index,
		epoch:           epoch,
		initPrivKey:     priKey,
		initPartPubKeys: initPartPubKeyP,
		threshold:       threshold,
		nParticipant:    nParticipant,
		suite:           suite,
	}

	err = dkgCrypt.createGenerator()
	if err != nil {
		log.Panicf("Can't create DKG generator epoch %d: %v", epoch, err)
	}

	return dkgCrypt
}

func (d *dkgCrypt) addInitPubKeys(pubKey kyber.Point) (bool, error) {
	if d.initPartPubKeys[0].Equal(pubKey) {
		pubHex, _ := encoding.PointToStringHex(d.suite, pubKey)
		d.log.Warnf("Receive my self init pub key %s", pubHex)
		return false, nil
	}

	d.partPubKeysSync.Lock()
	defer d.partPubKeysSync.Unlock()

	d.initPartPubKeys = append(d.initPartPubKeys, pubKey)

	pubHex, _ := encoding.PointToStringHex(d.suite, pubKey)
	d.log.Infof("Add participant pub hex %s", pubHex)

	if len(d.initPartPubKeys) == d.nParticipant {
		return true, nil
	}

	return false, nil
}

func (d *dkgCrypt) createGenerator() error {
	if d.dkGenerator != nil {
		d.log.Info("Dkg generator exist!")
		return nil
	}
	dGen, err := dkg.NewDistKeyGenerator(d.suite, d.initPrivKey, d.initPartPubKeys, d.threshold)
	if err != nil {
		d.log.Errorf("Create dkg generator faild: %v", err)
		return err
	}

	d.dkGenerator = dGen

	return nil
}

func (d *dkgCrypt) getEpoch() uint64 {
	return d.epoch
}

func (d *dkgCrypt) pubKey(index uint32) string {
	/*if index < 0 || index >= len(d.initPartPubKeys) {
		d.log.Panicf("Out of index: %d[0, %d)", index, len(d.initPartPubKeys))
	}*/
	pubKeyP, err := d.dkGenerator.GetParticipantPub(index)
	if err != nil {
		d.log.Panic(err.Error())
	}
	pkHex, _ := encoding.PointToStringHex(d.suite, pubKeyP /*d.initPartPubKeys[index]*/)

	return pkHex
}

func (d *dkgCrypt) dealSignVerifyPub(dealIndex uint32) string {
	pubKeyP, err := d.dkGenerator.GetDealVerifySignPub(dealIndex)
	if err != nil {
		d.log.Panic(err.Error())
	}
	pkHex, _ := encoding.PointToStringHex(d.suite, pubKeyP /*d.initPartPubKeys[index]*/)

	return pkHex
}

func (d *dkgCrypt) getDeals() (map[int]*dkg.Deal, error) {
	if d.dkGenerator == nil {
		errStr := "DKG generator hasn't been created"
		d.log.Error(errStr)
		return nil, errors.New(errStr)
	}

	return d.dkGenerator.Deals()
}

func (d *dkgCrypt) processDeal(deal *dkg.Deal) (*dkg.Response, error) {
	d.remoteDealsSync.Lock()
	defer d.remoteDealsSync.Unlock()

	d.log.Debugf("Process deal: %d", deal.Index)

	for _, dl := range d.remoteDeals {
		dlBytes, _ := dl.MarshalMsg(nil)
		dealBytes, _ := deal.MarshalMsg(nil)
		if bytes.Equal(dlBytes, dealBytes) {
			err := fmt.Errorf("Receive the same deal, index=%d: %v", deal.Index, deal)
			d.log.Warnf(err.Error())
			return nil, err
		}
	}

	d.remoteDeals = append(d.remoteDeals, deal)

	resp, err := d.dkGenerator.ProcessDeal(deal)

	if d.dkGenerator.ExpectedDeals() == d.nParticipant-1 {
		d.processAdvanceResp()
	}

	return resp, err
}

func (d *dkgCrypt) addAdvanceResp(resp *dkg.Response) error {
	d.remoteRespsSync.Lock()
	defer d.remoteRespsSync.Unlock()

	for _, res := range d.remoteAdvanResp {
		resBytes, _ := res.MarshalMsg(nil)
		respBytes, _ := resp.MarshalMsg(nil)
		if bytes.Equal(resBytes, respBytes) {
			err := fmt.Errorf("Receive the same resp, index=%d: %v", resp.Index, resp)
			d.log.Warnf(err.Error())
			return nil
		}
	}

	d.remoteAdvanResp = append(d.remoteAdvanResp, resp)

	return nil
}

func (d *dkgCrypt) processResp(resp *dkg.Response) error {
	d.log.Debugf("Process deal resp : index=%d, issIndex=%d", resp.Index, resp.Response.Index)

	j, err := d.dkGenerator.ProcessResponse(resp)
	if err != nil {
		return err
	}

	if j != nil {
		d.log.Warnf("Got justification %v from response %d", j, resp.Index)
	}

	return nil
}

func (d *dkgCrypt) processAdvanceResp() error {
	for _, resp := range d.remoteAdvanResp {
		j, err := d.dkGenerator.ProcessResponse(resp)

		if err != nil {
			if err == vss.ErrExistResponseOfSameOrigin {
				d.log.Warn("Process advance response existed same origin")
				continue
			} else {
				d.log.Warnf("Process advance response failed and will exit: %v", err)
				return err
			}
		}

		if j != nil {
			d.log.Warnf("Got justification %v from response %d when process dvance response", j, resp.Index)
		}
	}

	return nil
}

func (d *dkgCrypt) Finished() bool {
	return d.dkGenerator.Certified()
}

func (d *dkgCrypt) Sign(msg []byte) ([]byte, []byte, error) {
	if d.dkGenerator == nil {
		err := errors.New("Current state invalid and can't sign msg")
		d.log.Error(err.Error())

		return nil, nil, err
	}

	dkShare, err := d.dkGenerator.DistKeyShare()
	if err != nil {
		return nil, nil, err
	}

	pubKeyPoint := dkShare.Public()
	pubKey, err := pubKeyPoint.MarshalBinary()
	if err != nil {
		d.log.Errorf("Can't get pub key from point: %v", err)
		return nil, nil, err
	}

	signedInfo, err := tbls.Sign(d.suite, dkShare.PriShare(), msg)

	return signedInfo, pubKey, err
}

func (d *dkgCrypt) Verify(msg, sig []byte) error {
	if d.dkGenerator == nil {
		err := errors.New("Current state invalid and can't sign msg")
		d.log.Error(err.Error())

		return err
	}

	dkShare, err := d.dkGenerator.DistKeyShare()
	if err != nil {
		return err
	}

	pubPolicy := share.NewPubPoly(d.suite, d.suite.Point().Base(), dkShare.Commitments())
	if pubPolicy.Commit() != dkShare.Public() {
		err := errors.New("DKShare invalid: pub")
		d.log.Error(err.Error())
		return err
	}

	s := tbls.SigShare(sig)

	i, err := s.Index()
	if err != nil {
		return err
	}

	return bls.Verify(d.suite, pubPolicy.Eval(i).V, msg, s.Value())
}

func (d *dkgCrypt) RecoverSig(msg []byte, sigs [][]byte) ([]byte, error) {
	if d.dkGenerator == nil {
		err := errors.New("Current state invalid and can't sign msg")
		d.log.Error(err.Error())

		return nil, err
	}

	dkShare, err := d.dkGenerator.DistKeyShare()
	if err != nil {
		return nil, err
	}

	pubPoly := share.NewPubPoly(d.suite, d.suite.Point().Base(), dkShare.Commitments())

	return tbls.Recover(d.suite, pubPoly, msg, sigs, d.threshold, d.nParticipant)
}

func (d *dkgCrypt) Threshold() int {
	return d.threshold
}
