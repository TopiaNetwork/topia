package consensus

import (
	"bytes"
	"context"
	"sort"
	"sync"
	"sync/atomic"

	dkg "github.com/TopiaNetwork/kyber/v3/share/dkg/pedersen"
	vss "github.com/TopiaNetwork/kyber/v3/share/vss/pedersen"

	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	PreparePackedExeChannel_Size      = 10
	PreparePackedExeIndicChannel_Size = 10
	PreparePackedPropChannel_Size     = 10
	ProposeChannel_Size               = 10
	BestProposeChannel_Size           = 10

	PartPubKeyChannel_Size  = 500
	DealMSGChannel_Size     = 500
	DealRespMsgChannel_Size = 500
	FinishedMsgChannel_Size = 500
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
	chainID        types.ChainID
	nodeID         string
	startCh        chan uint64
	stopCh         chan struct{}
	finishedCh     chan bool
	partPubKey     chan *DKGPartPubKeyMessage
	dealMsgCh      chan *DKGDealMessage
	dealRespMsgCh  chan *DKGDealRespMessage
	finishedMsgCh  chan *DKGFinishedMessage
	deliver        dkgMessageDeliverI
	ledger         ledger.Ledger
	dkgExData      *dkgExchangeData
	dkgCrypt       *dkgCrypt
	updatersSync   sync.RWMutex
	dkgBLSUpdaters []DKGBLSUpdater
}

func newDKGExchange(log tplog.Logger,
	chainID types.ChainID,
	nodeID string,
	partPubKey chan *DKGPartPubKeyMessage,
	dealMsgCh chan *DKGDealMessage,
	dealRespMsgCh chan *DKGDealRespMessage,
	finishedMsgCh chan *DKGFinishedMessage,
	initDKGPrivKey string,
	deliver dkgMessageDeliverI,
	ledger ledger.Ledger) *dkgExchange {
	dkgExData := &dkgExchangeData{}
	dkgExData.initDKGPrivKey.Store(initDKGPrivKey)
	dkgExData.State.Store(DKGExchangeState_IDLE)

	return &dkgExchange{
		log:           tplog.CreateSubModuleLogger("DKG", log),
		chainID:       chainID,
		nodeID:        nodeID,
		startCh:       make(chan uint64, 1),
		stopCh:        make(chan struct{}, 1),
		finishedCh:    make(chan bool, 1),
		partPubKey:    partPubKey,
		dealMsgCh:     dealMsgCh,
		dealRespMsgCh: dealRespMsgCh,
		finishedMsgCh: finishedMsgCh,
		deliver:       deliver,
		ledger:        ledger,
		dkgExData:     dkgExData,
	}
}

func (ex *dkgExchange) updateDKGPrivKey(dkgPriKey string) {
	ex.dkgExData.initDKGPrivKey.Swap(dkgPriKey)
}

func (ex *dkgExchange) updateDKGPartPubKeys(dkgPartPubKeys map[string]string) error {
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

	sort.Strings(rtnPubKeys)

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

func (ex *dkgExchange) initWhenStart(triggerNumber uint64) {
	dkgPrivKey := ex.dkgExData.initDKGPrivKey.Load().(string)
	dkgPartPubKeys := ex.getDKGPartPubKeys()
	nParticipant := len(dkgPartPubKeys)
	dkgCrypt := newDKGCrypt(ex.log, triggerNumber, dkgPrivKey, dkgPartPubKeys, 2*nParticipant/3+1, nParticipant)
	ex.setDKGCrypt(dkgCrypt)
}

func (ex *dkgExchange) start(triggerNumber uint64, ctx context.Context) {
	ex.startCh <- triggerNumber

	ex.startReceiveDealLoop(ctx)

	ex.startReceiveDealRespLoop(ctx)

	ex.startReceiveFinishedMsg(ctx)
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
			case triggerNumber := <-ex.startCh:
				ex.log.Infof("DKG exchange send deal start %s", ex.nodeID)
				if ex.dkgCrypt == nil {
					ex.log.Panicf("dkgCrypt nil at present: trigger number %d", ex.dkgCrypt.triggerNumber)
				}

				if ex.dkgCrypt.triggerNumber != triggerNumber {
					ex.log.Panicf("dkgCrypt invalid expected triggerNumber=%d, got trigger number %d", triggerNumber, ex.dkgCrypt.triggerNumber)
				}

				deals, err := ex.dkgCrypt.getDeals()
				if err != nil {
					ex.log.Panicf("Can't get deals trigger number %d: %v", ex.dkgCrypt.triggerNumber, err)
				}

				for i, deal := range deals {
					dealBytes, err := deal.MarshalMsg(nil)
					if err != nil {
					}

					dealMsg := &DKGDealMessage{
						ChainID:       []byte(ex.chainID),
						Version:       CONSENSUS_VER,
						TriggerNumber: ex.dkgCrypt.triggerNumber,
						DealData:      dealBytes,
					}
					pubKey := ex.dkgCrypt.pubKey(uint32(i))
					destNodeID := ex.getNodeIDByDKGPartPubKey(pubKey)
					if destNodeID == "" {
						ex.log.Panicf("can't find the responding node ID by dkg part pub key: trigger number %d", ex.dkgCrypt.triggerNumber)
					}

					verifierPub, _ := ex.dkgCrypt.dkGenerator.GetVerifierPubOfDealer(i)
					ex.log.Infof("Send deal %d dealIndex %d from %s to other participant %s %s, enc verifier pub %s, expected signVerify pub %s, %v",
						i,
						deal.Index,
						ex.nodeID,
						destNodeID,
						pubKey,
						verifierPub.String(),
						ex.dkgCrypt.dealSignVerifyPub(deal.Index),
						ex.dkgExData.initDKGPartPubKeys.Load().(map[string]string))

					err = ex.deliver.deliverDKGDealMessage(ctx, destNodeID, dealMsg)
					if err != nil {
						ex.log.Panicf("Can't deliver deal trigger number %d: %v", ex.dkgCrypt.triggerNumber, err)
					}
				}
				ex.dkgExData.State.Swap(DKGExchangeState_Exchanging_Deal)
			case <-ex.stopCh:
				ex.log.Info("DKG exchange new trigger loop stop")
				return
			case <-ctx.Done():
				ex.log.Info("DKG exchange new trigger loop stop")
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
				var deal dkg.Deal
				_, err := deal.UnmarshalMsg(dealMsg.DealData)
				if err != nil {
					ex.log.Panicf("Can't unmarshal deal trigger number %d: %v", ex.dkgCrypt.triggerNumber, err)
					continue
				}

				ex.log.Infof("Node %s receive dealIndex %d trigger number %d", ex.nodeID, deal.Index, dealMsg.TriggerNumber)

				dealResp, err := ex.dkgCrypt.processDeal(&deal)
				if err != nil {
					if err == vss.ErrDealAlreadyProcessed {
						ex.log.Warnf("Process received deal dealIndex %d trigger number %d: self nodeID %s, expected signVerify pub %s, %v", deal.Index, ex.dkgCrypt.triggerNumber, ex.nodeID, ex.dkgCrypt.dealSignVerifyPub(deal.Index), ex.dkgExData.initDKGPartPubKeys.Load().(map[string]string))
						continue
					} else {
						ex.log.Panicf("Process deal failed dealIndex %d trigger number %d: %v, self nodeID %s, expected signVerify pub %s, %v", deal.Index, ex.dkgCrypt.triggerNumber, err, ex.nodeID, ex.dkgCrypt.dealSignVerifyPub(deal.Index), ex.dkgExData.initDKGPartPubKeys.Load().(map[string]string))
						continue
					}
				} else {
					ex.log.Infof("Node %s process deal successes dealIndex %d epoch %d, self nodeID %s, expected signVerify pub %s, %v", ex.nodeID, deal.Index, ex.dkgCrypt.triggerNumber, ex.nodeID, ex.dkgCrypt.dealSignVerifyPub(deal.Index), ex.dkgExData.initDKGPartPubKeys.Load().(map[string]string))
				}

				if dealResp.Response.Status != vss.StatusApproval {
					ex.log.Panicf("Deal response not approval dealIndex %d trigger number %d", deal.Index, ex.dkgCrypt.triggerNumber)
					continue
				}

				dealRespBytes, err := dealResp.MarshalMsg(nil)
				if err != nil {
					ex.log.Panicf("Marshal deal responses failed dealIndex %d trigger number %d: %v", deal.Index, ex.dkgCrypt.triggerNumber, err)
					continue
				}

				dealRespMsg := &DKGDealRespMessage{
					ChainID:       []byte(ex.chainID),
					Version:       CONSENSUS_VER,
					TriggerNumber: ex.dkgCrypt.triggerNumber,
					RespData:      dealRespBytes,
				}

				ex.log.Infof("Node %s send deal %d response to other participants", ex.nodeID, deal.Index)

				err = ex.deliver.deliverDKGDealRespMessage(ctx, dealRespMsg)
				if err != nil {
					ex.log.Panicf("Can't deliver deal trigger number %d: %v", ex.dkgCrypt.triggerNumber, err)
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
				var resp dkg.Response
				_, err := resp.UnmarshalMsg(dealRespMsg.RespData)
				if err != nil {
					ex.log.Panicf("Can't Unmarshal response trigger number %d: %v", ex.dkgCrypt.triggerNumber, err)
				}

				ex.log.Infof("Node %s receive deal response %d trigger number %d", ex.nodeID, resp.Index, dealRespMsg.TriggerNumber)

				err = ex.dkgCrypt.processResp(&resp)
				if err != nil {
					if err == vss.ErrNoDealBeforeResponse {
						ex.log.Warnf("Process no deal before response: epoch %d", ex.dkgCrypt.triggerNumber)
						ex.dkgCrypt.addAdvanceResp(&resp)
					} else if err == vss.ErrExistResponseOfSameOrigin {
						ex.log.Warnf("Process same origin response: trigger number %d", ex.dkgCrypt.triggerNumber)
					} else {
						ex.log.Panicf("Process response failed: trigger number %d: %v", ex.dkgCrypt.triggerNumber, err)
					}
				}

				ex.log.Infof("Node %s process deal response successed %d trigger number %d", ex.nodeID, resp.Index, dealRespMsg.TriggerNumber)

				if ex.dkgCrypt.Finished() {
					ex.log.Infof("DKG exchange finished: self node %s", ex.nodeID)
					ex.dkgExData.State.Swap(DKGExchangeState_Finished)
					ex.finishedCh <- true

					pubPolicyComm, err := ex.dkgCrypt.PubKey()
					if err != nil {
						ex.log.Panicf("Can't get public policy commit: trigger number %d", ex.dkgCrypt.triggerNumber)
					}

					finishedMsg := &DKGFinishedMessage{
						ChainID:       []byte(ex.chainID),
						Version:       CONSENSUS_VER,
						TriggerNumber: ex.dkgCrypt.triggerNumber,
						PubPolyCommit: pubPolicyComm,
					}

					ex.log.Infof("Node send finished message to other participants: trigger number %d, self node %s", ex.dkgCrypt.triggerNumber, ex.nodeID)

					err = ex.deliver.deliverDKGFinishedMessage(ctx, finishedMsg)
					if err != nil {
						ex.log.Panicf("Can't deliver finished message: %d, trigger number %d", err, ex.dkgCrypt.triggerNumber)
					}
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

func (ex *dkgExchange) startReceiveFinishedMsg(ctx context.Context) {
	go func() {
		for {
			select {
			case finishedMsg := <-ex.finishedMsgCh:
				ex.log.Infof("Node %s receive deal response %d trigger number %d", ex.nodeID, finishedMsg.TriggerNumber)

				if !ex.dkgCrypt.Finished() {
					func() {
						ex.dkgCrypt.remoteFinishedSync.Lock()
						defer ex.dkgCrypt.remoteFinishedSync.Unlock()

						ex.dkgCrypt.remoteFinished = append(ex.dkgCrypt.remoteFinished, finishedMsg)
					}()

					continue
				}

				selfPubPolicyCommt, err := ex.dkgCrypt.PubKey()
				if err != nil {
					ex.log.Errorf("Can't get public policy commit: %v, trgger number %d, self node %s", err, finishedMsg.TriggerNumber, ex.nodeID)
					continue
				}
				if !bytes.Equal(selfPubPolicyCommt, finishedMsg.PubPolyCommit) {
					ex.log.Errorf("Received invalid finished message: trgger number %d, self node %s", finishedMsg.TriggerNumber, ex.nodeID)
					continue
				}

				func() {
					ex.dkgCrypt.remoteFinishedSync.Lock()
					defer ex.dkgCrypt.remoteFinishedSync.Unlock()

					ex.dkgCrypt.remoteFinished = append(ex.dkgCrypt.remoteFinished, finishedMsg)
				}()

				if len(ex.dkgCrypt.remoteFinished) >= ex.dkgCrypt.nParticipant-1 {
					ex.updateDKGState(DKGExchangeState_IDLE)
					ex.notifyUpdater()
				}
			case <-ex.stopCh:
				ex.log.Info("DKG exchange receive finished message loop stop")
				return
			case <-ctx.Done():
				ex.log.Info("DKG exchange receive finished message loop stop")
				return
			}
		}
	}()
}

func (ex *dkgExchange) startLoop(ctx context.Context) {
	ex.log.Info("Start DKG exchange loop")
	ex.startSendDealLoop(ctx)
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
