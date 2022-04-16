package consensus

import (
	"context"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/common"
	"time"

	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

type epochService struct {
	log                 tplog.Logger
	nodeID              string
	blockAddedCh        chan *tpchaintypes.Block
	epochInterval       uint64 //the height number between two epochs
	dkgStartBeforeEpoch uint64 //the starting height number of DKG before an epoch
	exeScheduler        execution.ExecutionScheduler
	ledger              ledger.Ledger
	dkgExchange         *dkgExchange
}

func newEpochService(log tplog.Logger,
	nodeID string,
	blockAddedCh chan *tpchaintypes.Block,
	epochInterval uint64,
	dkgStartBeforeEpoch uint64,
	exeScheduler execution.ExecutionScheduler,
	ledger ledger.Ledger,
	dkgExchange *dkgExchange) *epochService {
	if ledger == nil || dkgExchange == nil {
		panic("Invalid input parameter and can't create epoch service!")
	}

	return &epochService{
		log:                 log,
		nodeID:              nodeID,
		blockAddedCh:        blockAddedCh,
		epochInterval:       epochInterval,
		dkgStartBeforeEpoch: dkgStartBeforeEpoch,
		exeScheduler:        exeScheduler,
		ledger:              ledger,
		dkgExchange:         dkgExchange,
	}
}

func (es *epochService) start(ctx context.Context) {
	go func() {
		for {
			select {
			case newBh := <-es.blockAddedCh:
				es.log.Infof("Epoch service receive new block added event: new block height %d", newBh.Head.Height)
				maxStateVer, err := es.exeScheduler.MaxStateVersion(es.log, es.ledger)
				if err != nil {
					es.log.Errorf("Can't get max state version: %v", err)
					continue
				}
				csStateEpoch := state.GetStateBuilder().CreateCompositionState(es.log, es.nodeID, es.ledger, maxStateVer+1)

				epochInfo, err := csStateEpoch.GetLatestEpoch()
				if err != nil {
					es.log.Errorf("Can't get the latest epoch: err=%v", err)
					continue
				}

				deltaH := int(newBh.Head.Height) - int(epochInfo.StartHeight)
				if deltaH == int(es.epochInterval)-int(es.dkgStartBeforeEpoch) {
					dkgExState := es.dkgExchange.getDKGState()
					if dkgExState != DKGExchangeState_IDLE {
						es.log.Warnf("Unfinished dkg exchange is going on an new dkg can't start, current state: %s", dkgExState.String)
					} else {
						err = es.dkgExchange.updateDKGPartPubKeys(csStateEpoch)
						if err != nil {
							es.log.Errorf("Update DKG exchange part pub keys err: %v", err)
							continue
						}
						es.dkgExchange.initWhenStart(epochInfo.Epoch + 1)
						es.dkgExchange.start(epochInfo.Epoch + 1)
					}
				} else if deltaH == int(es.epochInterval) {
					csStateEpoch.SetLatestEpoch(&common.EpochInfo{
						Epoch:          epochInfo.Epoch + 1,
						StartTimeStamp: uint64(time.Now().UnixNano()),
						StartHeight:    newBh.Head.Height,
					})

					if !es.dkgExchange.dkgCrypt.finished() {
						es.log.Warnf("Current epoch %d DKG unfinished and still use the old, height %d", epochInfo.Epoch, newBh.Head.Height)
					} else {
						es.dkgExchange.notifyUpdater()
						es.dkgExchange.updateDKGState(DKGExchangeState_IDLE)
					}
				}
			case <-ctx.Done():
				es.log.Info("Exit epoch cycle")
				return
			}
		}
	}()
}
