package consensus

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/TopiaNetwork/kyber/v3"
	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/share/dkg/pedersen"
	"github.com/TopiaNetwork/kyber/v3/util/encoding"
	tplog "github.com/TopiaNetwork/topia/log"
)

type DKGCrypt struct {
	log             tplog.Logger
	index           uint32
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

func newDKGCrypt(log tplog.Logger, index uint32, epoch uint64, suite *bn256.Suite, initPrivKey string, initPartPubKeys []string, threshold int, nParticipant int) *DKGCrypt {
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
			log.Panicf("Invalid initPubKey %s", initPartPubKeys[i])
		}
		initPartPubKeyP[i] = partPubKey
	}

	dkgCrypt := &DKGCrypt{
		log:             log,
		index:           index,
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

func (d *DKGCrypt) addInitPubKeys(pubKey kyber.Point) (bool, error) {
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

func (d *DKGCrypt) createGenerator() error {
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

func (d *DKGCrypt) getEpoch() uint64 {
	return d.epoch
}

func (d *DKGCrypt) pubKey(index int) string {
	if index < 0 || index >= len(d.initPartPubKeys) {
		d.log.Panicf("Out of index: %d[0, %d)", index, len(d.initPartPubKeys))
	}
	pkHex, _ := encoding.PointToStringHex(d.suite, d.initPartPubKeys[index])

	return pkHex
}

func (d *DKGCrypt) getDeals() (map[int]*dkg.Deal, error) {
	if d.dkGenerator == nil {
		errStr := "DKG generator hasn't been created"
		d.log.Error(errStr)
		return nil, errors.New(errStr)
	}

	return d.dkGenerator.Deals()
}

func (d *DKGCrypt) processDeal(deal *dkg.Deal) (*dkg.Response, error) {
	d.remoteDealsSync.Lock()
	defer d.remoteDealsSync.Unlock()

	for _, dl := range d.remoteDeals {
		dlBytes, _ := dl.MarshalMsg(nil)
		dealBytes, _ := deal.MarshalMsg(nil)
		if bytes.Equal(dlBytes, dealBytes) {
			err := fmt.Errorf("Receive the same deal, index=%d: %v", deal.Index, deal)
			d.log.Warnf(err.Error())
			return nil, err
		} else {
			d.remoteDeals = append(d.remoteDeals, deal)
		}
	}

	resp, err := d.dkGenerator.ProcessDeal(deal)

	if d.dkGenerator.ExpectedDeals() == d.nParticipant-1 {
		d.processAdvanceResp()
	}

	return resp, err
}

func (d *DKGCrypt) addAdvanceResp(resp *dkg.Response) error {
	d.remoteRespsSync.Lock()
	defer d.remoteRespsSync.Unlock()

	for _, res := range d.remoteAdvanResp {
		resBytes, _ := res.MarshalMsg(nil)
		respBytes, _ := resp.MarshalMsg(nil)
		if bytes.Equal(resBytes, respBytes) {
			err := fmt.Errorf("Receive the same resp, index=%d: %v", resp.Index, resp)
			d.log.Warnf(err.Error())
			return nil
		} else {
			d.remoteAdvanResp = append(d.remoteAdvanResp, resp)
		}
	}

	if d.remoteAdvanResp == nil {
		d.remoteAdvanResp = append(d.remoteAdvanResp, resp)
	}

	return nil
}

func (d *DKGCrypt) processResp(resp *dkg.Response) error {
	j, err := d.dkGenerator.ProcessResponse(resp)
	if err != nil {
		return err
	}

	if j != nil {
		d.log.Warnf("Got justification %v from response %d", j, resp.Index)
	}

	return nil
}

func (d *DKGCrypt) processAdvanceResp() error {
	for _, resp := range d.remoteAdvanResp {
		j, err := d.dkGenerator.ProcessResponse(resp)
		if err != nil {
			d.log.Errorf("Process advance response failed: %v", err)
			return err
		}

		if j != nil {
			d.log.Warnf("Got justification %v from response %d when process dvance response", j, resp.Index)
		}
	}

	return nil
}

func (d *DKGCrypt) finished() bool {
	return d.dkGenerator.Certified()
}
