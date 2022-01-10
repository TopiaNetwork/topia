package consensus

import (
	"context"

	dkg "github.com/TopiaNetwork/kyber/v3/share/dkg/pedersen"
	vss "github.com/TopiaNetwork/kyber/v3/share/vss/pedersen"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	PartPubKeyChannel_Size  = 150
	DealMSGChannel_Size     = 150
	DealRespMsgChannel_Size = 150
)

type dkgExchange struct {
	index         int
	log           tplog.Logger
	startCh       chan uint64
	stopCh        chan struct{}
	finished      chan bool
	partPubKey    chan *DKGPartPubKeyMessage
	dealMsgCh     chan *DKGDealMessage
	dealRespMsgCh chan *DKGDealRespMessage
	deliver       messageDeliverI
	csState       consensusStore
	dkgCrypt      *DKGCrypt
}

func newDKGExchange(log tplog.Logger,
	partPubKey chan *DKGPartPubKeyMessage,
	dealMsgCh chan *DKGDealMessage,
	dealRespMsgCh chan *DKGDealRespMessage,
	deliver messageDeliverI,
	csState consensusStore) *dkgExchange {
	return &dkgExchange{
		log:           log,
		startCh:       make(chan uint64),
		stopCh:        make(chan struct{}),
		finished:      make(chan bool),
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

func (ex *dkgExchange) startNewEpochLoop(ctx context.Context) {
	go func() {
		for {
			select {
			case epoch := <-ex.startCh:
				ex.log.Infof("DKG exchange start %d", ex.index)
				if ex.dkgCrypt == nil {
					ex.log.Panicf("dkgCrypt nil at present: epoch=%d")
				}

				if ex.dkgCrypt.epoch != epoch {
					ex.log.Panicf("dkgCrypt invalid expected epoch=%d, got epoch =%d", epoch, ex.dkgCrypt.epoch)
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
					pubKey := ex.dkgCrypt.pubKey(i)
					ex.log.Infof("Send deal %d to other participant %s", i, pubKey)

					err = ex.deliver.deliverDKGDealMessage(ctx, pubKey, dealMsg)
					if err != nil {
						ex.log.Panicf("Can't marshal deal epoch %d: %v", ex.dkgCrypt.epoch, err)
					}
				}
			case <-ex.stopCh:
				ex.log.Info("DKG exchange new epoch loop stop")
				return
			case <-ctx.Done():
				ex.log.Info("DKG exchange new epoch loop stop")
				return
			}
		}
	}()
}

func (ex *dkgExchange) startReceiveDealLoop(ctx context.Context) {
	go func() {
		for {
			select {
			case dealMsg := <-ex.dealMsgCh:
				ex.log.Infof("DKG exchange receive deal %d epoch=%d", ex.index, dealMsg.Epoch)

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
				} else {
					ex.log.Infof("Process deal succeeded dealIndex %d epoch %d", deal.Index, ex.dkgCrypt.epoch)
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
			case <-ex.stopCh:
				ex.log.Info("DKG exchange receive deal loop stop")
				return
			case <-ctx.Done():
				ex.log.Info("DKG exchange receive deal loop stop")
				return
			}
		}
	}()
}

func (ex *dkgExchange) startReceiveDealRespLoop(ctx context.Context) {
	go func() {
		for {
			select {
			case dealRespMsg := <-ex.dealRespMsgCh:
				ex.log.Infof("DKG exchange receive deal response %d epoch=%d", ex.index, dealRespMsg.Epoch)

				var resp dkg.Response
				_, err := resp.UnmarshalMsg(dealRespMsg.RespData)
				if err != nil {
					ex.log.Errorf("Can't Unmarshal response epoch %d: %v", ex.dkgCrypt.epoch, err)
				}
				err = ex.dkgCrypt.processResp(&resp)
				if err != nil {
					if err == vss.ErrNoDealBeforeResponse {
						ex.log.Warnf("Process response failed: epoch %d: %v", ex.dkgCrypt.epoch, err)
						ex.dkgCrypt.addAdvanceResp(&resp)
					} else {
						ex.log.Errorf("Process response failed: epoch %d: %v", ex.dkgCrypt.epoch, err)
					}
				}

				if ex.dkgCrypt.finished() {
					ex.log.Info("DKG exchange finished")
					ex.finished <- true
				}
			case <-ex.stopCh:
				ex.log.Info("DKG exchange receive deal response loop stop")
				return
			case <-ctx.Done():
				ex.log.Info("DKG exchange receive deal response loop stop")
				return
			}
		}
	}()
}

func (ex *dkgExchange) startLoop(ctx context.Context) {
	ex.log.Info("Start DKG exchange loop")
	ex.startNewEpochLoop(ctx)
	ex.startReceiveDealLoop(ctx)
	ex.startReceiveDealRespLoop(ctx)
}
