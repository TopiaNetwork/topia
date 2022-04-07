package consensus

import (
	"context"
	"sync"
	"sync/atomic"

	dkg "github.com/TopiaNetwork/kyber/v3/share/dkg/pedersen"
	vss "github.com/TopiaNetwork/kyber/v3/share/vss/pedersen"

	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

const (
	PartPubKeyChannel_Size  = 150
	DealMSGChannel_Size     = 150
	DealRespMsgChannel_Size = 150
)

type DKGExchangeState byte

const (
	DKGExchangeState_Unknown DKGExchangeState = iota
	DKGExchangeState_IDLE
	DKGExchangeState_Exchanging_PartPubKey //Haven't used it at present
	DKGExchangeState_Exchanging_Deal
	DKGExchangeState_Finished
)

type dkgExchangeData struct {
	initDKGPrivKey     atomic.Value //string
	initDKGPartPubKeys atomic.Value //map[string]string. nodeID->dkgPartPubKey
	State              atomic.Value //DKGExchangeState
}

type dkgExchange struct {
	index          int
	log            tplog.Logger
	nodeID         string
	startCh        chan uint64
	stopCh         chan struct{}
	finishedCh     chan bool
	partPubKey     chan *DKGPartPubKeyMessage
	dealMsgCh      chan *DKGDealMessage
	dealRespMsgCh  chan *DKGDealRespMessage
	deliver        messageDeliverI
	ledger         ledger.Ledger
	dkgExData      *dkgExchangeData
	dkgCrypt       *dkgCrypt
	updatersSync   sync.RWMutex
	dkgBLSUpdaters []DKGBLSUpdater
}

func newDKGExchange(log tplog.Logger,
	nodeID string,
	partPubKey chan *DKGPartPubKeyMessage,
	dealMsgCh chan *DKGDealMessage,
	dealRespMsgCh chan *DKGDealRespMessage,
	initDKGPrivKey string,
	deliver messageDeliverI,
	ledger ledger.Ledger) *dkgExchange {
	dkgExData := &dkgExchangeData{}
	dkgExData.initDKGPrivKey.Store(initDKGPrivKey)
	dkgExData.State.Store(DKGExchangeState_IDLE)

	return &dkgExchange{
		log:           log,
		nodeID:        nodeID,
		startCh:       make(chan uint64),
		stopCh:        make(chan struct{}),
		finishedCh:    make(chan bool),
		partPubKey:    partPubKey,
		dealMsgCh:     dealMsgCh,
		dealRespMsgCh: dealRespMsgCh,
		deliver:       deliver,
		ledger:        ledger,
		dkgExData:     dkgExData,
	}
}

func (ex *dkgExchange) updateDKGPrivKey(dkgPriKey string) {
	ex.dkgExData.initDKGPrivKey.Swap(dkgPriKey)
}

func (ex *dkgExchange) updateDKGPartPubKeys(compStateRN state.CompositionStateReadonly) error {
	dkgPartPubKeys, err := compStateRN.GetDKGPartPubKeysForVerify()
	if err != nil {
		return err
	}

	ex.dkgExData.initDKGPartPubKeys.Swap(dkgPartPubKeys)

	return nil
}

func (ex *dkgExchange) updateDKGState(exchangeState DKGExchangeState) {
	ex.dkgExData.State.Swap(exchangeState)
}

func (ex *dkgExchange) getDKGPrivKey() string {
	return ex.dkgExData.initDKGPrivKey.Load().(string)
}

func (ex *dkgExchange) getDKGPartPubKeys() []string {
	dkgPartPubKeys := ex.dkgExData.initDKGPartPubKeys.Load().(map[string]string)
	var rtnPubKeys []string
	for _, pubKey := range dkgPartPubKeys {
		rtnPubKeys = append(rtnPubKeys, pubKey)
	}

	return rtnPubKeys
}

func (ex *dkgExchange) getDKGState() DKGExchangeState {
	return ex.dkgExData.State.Load().(DKGExchangeState)
}

func (ex *dkgExchange) setDKGCrypt(dkgCrypt *dkgCrypt) {
	ex.dkgCrypt = dkgCrypt
}

func (ex *dkgExchange) addDKGBLSUpdater(updater DKGBLSUpdater) {
	ex.updatersSync.Lock()
	defer ex.updatersSync.Unlock()

	ex.dkgBLSUpdaters = append(ex.dkgBLSUpdaters, updater)
}

func (ex *dkgExchange) notifyUpdater() {
	ex.updatersSync.RLock()
	defer ex.updatersSync.RUnlock()
	for _, updater := range ex.dkgBLSUpdaters {
		updater.updateDKGBls(ex.dkgCrypt)
	}
}

func (ex *dkgExchange) start(epoch uint64) {
	dkgPrivKey := ex.dkgExData.initDKGPrivKey.Load().(string)
	dkgPartPubKeys := ex.getDKGPartPubKeys()
	nParticipant := len(dkgPartPubKeys)
	dkgCrypt := newDKGCrypt(ex.log, epoch, dkgPrivKey, dkgPartPubKeys, 2*nParticipant/3+1, nParticipant)
	ex.setDKGCrypt(dkgCrypt)
	ex.startCh <- epoch
}

func (ex *dkgExchange) stop() {
	ex.stopCh <- struct{}{}
}

func (ex *dkgExchange) getNodeIDByDKGPartPubKey(dkgPartPubKey string) string {
	dkgPartPubKeys := ex.dkgExData.initDKGPartPubKeys.Load().(map[string]string)
	for nodeID, pubKey := range dkgPartPubKeys {
		if pubKey == dkgPartPubKey {
			return nodeID
		}
	}

	return ""
}

func (ex *dkgExchange) startSendDealLoop(ctx context.Context) {
	go func() {
		for {
			select {
			case epoch := <-ex.startCh:
				csStateRN := state.CreateCompositionStateReadonly(ex.log, ex.ledger)
				defer csStateRN.Stop()

				ex.log.Infof("DKG exchange start %s", ex.nodeID)
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
						ChainID:  []byte(csStateRN.ChainID()),
						Version:  CONSENSUS_VER,
						Epoch:    ex.dkgCrypt.epoch,
						DealData: dealBytes,
					}
					pubKey := ex.dkgCrypt.pubKey(i)
					destNodeID := ex.getNodeIDByDKGPartPubKey(pubKey)
					if destNodeID == "" {
						ex.log.Panicf("can't find the responding node ID by dkg part pub key: epoch=%d", ex.dkgCrypt.epoch)
					}
					ex.log.Infof("Send deal %d to other participant %s", i, destNodeID)

					err = ex.deliver.deliverDKGDealMessage(ctx, destNodeID, dealMsg)
					if err != nil {
						ex.log.Panicf("Can't marshal deal epoch %d: %v", ex.dkgCrypt.epoch, err)
					}
				}
				ex.dkgExData.State.Swap(DKGExchangeState_Exchanging_Deal)
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
				csStateRN := state.CreateCompositionStateReadonly(ex.log, ex.ledger)
				defer csStateRN.Stop()

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
					ChainID:  []byte(csStateRN.ChainID()),
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
					ex.dkgExData.State.Swap(DKGExchangeState_Finished)
					ex.finishedCh <- true
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
	ex.startSendDealLoop(ctx)
	ex.startReceiveDealLoop(ctx)
	ex.startReceiveDealRespLoop(ctx)
}

func (state DKGExchangeState) String() string {
	switch state {
	case DKGExchangeState_IDLE:
		return "Idle"
	case DKGExchangeState_Exchanging_PartPubKey:
		return "ExchangingPartPubKey"
	case DKGExchangeState_Exchanging_Deal:
		return "ExchangingDeal"
	case DKGExchangeState_Finished:
		return "Finished"
	default:
		return "Unknown"
	}
}

func (state DKGExchangeState) Value(stas string) DKGExchangeState {
	switch stas {
	case "Idle":
		return DKGExchangeState_IDLE
	case "ExchangingPartPubKey":
		return DKGExchangeState_Exchanging_PartPubKey
	case "ExchangingDeal":
		return DKGExchangeState_Exchanging_Deal
	case "Finished":
		return DKGExchangeState_Finished
	default:
		return DKGExchangeState_Unknown
	}
}
