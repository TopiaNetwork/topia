package consensus

import (
	"errors"
	"github.com/TopiaNetwork/kyber/v3/util/encoding"
	"github.com/TopiaNetwork/kyber/v3/util/key"
	"sync"

	"github.com/TopiaNetwork/kyber/v3"
	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/share/dkg/pedersen"
	tplog "github.com/TopiaNetwork/topia/log"
)

type DKGCrypt struct {
	log             tplog.Logger
	epoch           uint64
	initPrivKey     kyber.Scalar
	partPubKeysSync sync.RWMutex
	initPartPubKeys []kyber.Point
	threshold       int
	nParticipant    int
	suite           *bn256.Suite
	remoteDealsSync sync.RWMutex
	remoteDeals     map[uint32][]*dkg.Deal
	dkGenerator     *dkg.DistKeyGenerator
}

func newDKGCrypt(log tplog.Logger, epoch uint64, threshold int, nParticipant int) *DKGCrypt {
	suite := bn256.NewSuiteG2()

	keyP := key.NewKeyPair(suite)

	return &DKGCrypt{
		log:             log,
		epoch:           epoch,
		initPrivKey:     keyP.Private,
		initPartPubKeys: []kyber.Point{keyP.Public},
		threshold:       threshold,
		nParticipant:    nParticipant,
		suite:           suite,
		remoteDeals:     make(map[uint32][]*dkg.Deal),
	}
}

func (d *DKGCrypt) addInitPubKey(pubKey kyber.Point) (bool, error) {
	if d.initPartPubKeys[0].Equal(pubKey) {
		pubHex, _ := encoding.PointToStringHex(d.suite, pubKey)
		d.log.Warnf("Receive my self init pub key %s", pubHex)
		return false, nil
	}

	d.partPubKeysSync.Lock()
	defer d.partPubKeysSync.Unlock()

	d.initPartPubKeys = append(d.initPartPubKeys, pubKey)

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

	d.remoteDeals[deal.Index] = append(d.remoteDeals[deal.Index], deal)

	return d.dkGenerator.ProcessDeal(deal)
}

func (d *DKGCrypt) processResp(resp *dkg.Response) error {
	j, err := d.dkGenerator.ProcessResponse(resp)
	if err != nil {
		d.log.Errorf("Process response failed: %v", err)
		return err
	}

	if j != nil {
		d.log.Warnf("Got justification %v from response %d", j, resp.Index)
	}

	return nil
}
