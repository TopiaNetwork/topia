package consensus

import (
	"context"

	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	dkg "github.com/TopiaNetwork/kyber/v3/share/dkg/pedersen"
	vss "github.com/TopiaNetwork/kyber/v3/share/vss/pedersen"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	DealMSGChannel_Size     = 150
	DealRespMsgChannel_Size = 150
)

type dkgExchange struct {
	log           tplog.Logger
	startCh       chan uint64
	stopCh        chan struct{}
	partPubKey    chan *DKGPartPubKeyMessage
	dealMsgCh     chan *DKGDealMessage
	dealRespMsgCh chan *DKGDealRespMessage
	deliver       *messageDeliver
	csState       ConsensusStore
	dkgCrypt      *DKGCrypt
}

func newDKGExchange(log tplog.Logger,
	partPubKey chan *DKGPartPubKeyMessage,
	dealMsgCh chan *DKGDealMessage,
	dealRespMsgCh chan *DKGDealRespMessage,
	deliver *messageDeliver,
	csState ConsensusStore) *dkgExchange {
	return &dkgExchange{
		log:           log,
		startCh:       make(chan uint64),
		stopCh:        make(chan struct{}),
		partPubKey:    partPubKey,
		dealMsgCh:     dealMsgCh,
		dealRespMsgCh: dealRespMsgCh,
		deliver:       deliver,
		csState:       csState,
	}
}

func (ex *dkgExchange) setDKGCrypt(dkgCrypt *DKGCrypt) {
	ex.dkgCrypt = dkgCrypt
}

func (ex *dkgExchange) start(epoch uint64) {
	ex.startCh <- epoch
}

func (ex *dkgExchange) stop() {
	ex.stopCh <- struct{}{}
}

func (ex *dkgExchange) startLoop(ctx context.Context) {
	go func() {
		ex.log.Info("Start DKG exchange loop")
		defer ex.log.Info("End DKG exchange loop")
		for {
			select {
			case epoch := <-ex.startCh:
				ex.log.Info("DKG exchange start")
				if ex.dkgCrypt == nil {
					ex.log.Panicf("dkgCrypt nil at present: epoch=%d")
				}

				if ex.dkgCrypt.epoch != epoch {
					ex.log.Panicf("dkgCrypt invalid expected epoch=%d, got epoch =%d", epoch, ex.dkgCrypt.epoch)
				}

				initPubKeyBytes, err := ex.dkgCrypt.initPartPubKeys[0].MarshalBinary()
				if err != nil {
					ex.log.Panicf("Can't marshal pub key %v epoch=%d: err=%v", ex.dkgCrypt.initPartPubKeys[0], epoch, err)
				}

				partPubKeyMsg := &DKGPartPubKeyMessage{
					ChainID:    []byte(ex.csState.ChainID()),
					Version:    CONSENSUS_VER,
					Epoch:      ex.dkgCrypt.epoch,
					PartPubKey: initPubKeyBytes,
				}
				err = ex.deliver.deliverDKGPartPubKeyMessage(ctx, partPubKeyMsg)
				if err != nil {
					ex.log.Panicf("Can't deliver DKG part pub key msg deal epoch %d: %v", ex.dkgCrypt.epoch, err)
				}
			case partPubKey := <-ex.partPubKey:
				pubKey, err := bn256.UnmarshalBinaryPG2(partPubKey.PartPubKey)
				if err != nil {
					ex.log.Panicf("Can't unmarshal part pub key epoch %d: %v", ex.dkgCrypt.epoch, err)
				}

				ok, err := ex.dkgCrypt.addInitPubKey(pubKey)
				if err != nil {
					ex.log.Panicf("Can't add initial pub key epoch %d: %v", ex.dkgCrypt.epoch, err)
				}
				if !ok {
					continue
				} else {
					err := ex.dkgCrypt.createGenerator()
					if err != nil {
						ex.log.Panicf("Can't create generator epoch %d: %v", ex.dkgCrypt.epoch, err)
					}

					deals, err := ex.dkgCrypt.getDeals()
					if err != nil {
						ex.log.Panicf("Can't get deals epoch %d: %v", ex.dkgCrypt.epoch, err)
					}

					for i, deal := range deals {
						dealBytes, err := deal.MarshalMsg(nil)
						if err != nil {
						}

						dealMsg := &DKGDealMessage{
							ChainID:  []byte(ex.csState.ChainID()),
							Version:  CONSENSUS_VER,
							Epoch:    ex.dkgCrypt.epoch,
							DealData: dealBytes,
						}

						ex.log.Infof("Send deal %d to other participants", i)

						err = ex.deliver.deliverDKGDealMessage(ctx, dealMsg)
						if err != nil {
							ex.log.Panicf("Can't marshal deal epoch %d: %v", ex.dkgCrypt.epoch, err)
						}
					}
				}
			case dealMsg := <-ex.dealMsgCh:
				ex.log.Infof("DKG exchange receive deal epoch=%d", dealMsg.Epoch)

				var deal dkg.Deal
				_, err := deal.UnmarshalMsg(dealMsg.DealData)
				if err != nil {
					ex.log.Errorf("Can't unmarshal deal epoch %d: %v", ex.dkgCrypt.epoch, err)
					continue
				}

				dealResp, err := ex.dkgCrypt.processDeal(&deal)
				if err != nil {
					ex.log.Errorf("Process deal failed dealIndex %d epoch %d: %v", deal.Index, ex.dkgCrypt.epoch, err)
					continue
				}
				if dealResp.Response.Status != vss.StatusApproval {
					ex.log.Errorf("Deal response not approval dealIndex %d epoch %d", deal.Index, ex.dkgCrypt.epoch)
					continue
				}

				dealRespBytes, err := dealResp.MarshalMsg(nil)
				if err != nil {
					ex.log.Errorf("Marshal deal responsse failed dealIndex %d epoch %d: %v", deal.Index, ex.dkgCrypt.epoch, err)
					continue
				}

				dealRespMsg := &DKGDealRespMessage{
					ChainID:  []byte(ex.csState.ChainID()),
					Version:  CONSENSUS_VER,
					Epoch:    ex.dkgCrypt.epoch,
					RespData: dealRespBytes,
				}

				ex.log.Infof("Send deal %d response to other participants", deal.Index)

				err = ex.deliver.deliverDKGDealRespMessage(ctx, dealRespMsg)
				if err != nil {
					ex.log.Errorf("Can't marshal deal epoch %d: %v", ex.dkgCrypt.epoch, err)
				}
			case dealRespMsg := <-ex.dealRespMsgCh:
				ex.log.Infof("DKG exchange receive deal response epoch=%d", dealRespMsg.Epoch)

				var resp dkg.Response
				_, err := resp.UnmarshalMsg(dealRespMsg.RespData)
				if err != nil {
					ex.log.Errorf("Can't Unmarshal response epoch %d: %v", ex.dkgCrypt.epoch, err)
				}
				err = ex.dkgCrypt.processResp(&resp)
				if err != nil {
					ex.log.Errorf("Process response failed: epoch %d: %v", ex.dkgCrypt.epoch, err)
				}
			case <-ex.stopCh:
				ex.log.Info("DKG exchange stop")
				return
			case <-ctx.Done():
				ex.log.Info("DKG exchange stop")
				return
			}
		}
	}()
}
