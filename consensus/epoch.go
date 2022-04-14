package consensus

import (
	"context"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

type epochService struct {
	log                 tplog.Logger
	nodeID              string
	blockAddedCh        chan *tpchaintypes.Block
	epochNewCh          chan *tpcmm.EpochInfo
	epochInterval       uint64 //the height number between two epochs
	dkgStartBeforeEpoch uint64 //the starting height number of DKG before an epoch
	exeScheduler        execution.ExecutionScheduler
	ledger              ledger.Ledger
	dkgExchange         *dkgExchange
}

func newEpochService(log tplog.Logger,
	nodeID string,
	blockAddedCh chan *tpchaintypes.Block,
	epochNewCh chan *tpcmm.EpochInfo,
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
		epochNewCh:          epochNewCh,
		epochInterval:       epochInterval,
		dkgStartBeforeEpoch: dkgStartBeforeEpoch,
		exeScheduler:        exeScheduler,
		ledger:              ledger,
		dkgExchange:         dkgExchange,
	}
}

func (es *epochService) processBlockAddedStart(ctx context.Context) {
	go func() {
		for {
			select {
			case newBh := <-es.blockAddedCh:
				err := func() error {
					es.log.Infof("Epoch services receive new block added event: new block height %d", newBh.Head.Height)
					csStateEpoch := state.CreateCompositionStateReadonly(es.log, es.ledger)
					defer csStateEpoch.Stop()

					epochInfo, err := csStateEpoch.GetLatestEpoch()
					if err != nil {
						es.log.Errorf("Can't get the latest epoch: err=%v", err)
						return err
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
								return err
							}
							es.dkgExchange.initWhenStart(epochInfo.Epoch + 1)
							es.dkgExchange.start(epochInfo.Epoch + 1)
						}
					}

					return nil
				}()

				if err != nil {
					continue
				}
			case <-ctx.Done():
				es.log.Info("Epoch service propcess block added exit")
				return
			}
		}
	}()
}

func (es *epochService) processEpochNewStart(ctx context.Context) {
	go func() {
		for {
			select {
			case newEpoch := <-es.epochNewCh:

				es.log.Infof("Epoch service receives new epoch event: new epoch %d", newEpoch.Epoch)
				if !es.dkgExchange.dkgCrypt.finished() {
					es.log.Warnf("Current epoch %d DKG unfinished and still use the old, height %d", newEpoch.Epoch, newEpoch.StartHeight)
				} else {
					es.dkgExchange.notifyUpdater()
					es.dkgExchange.updateDKGState(DKGExchangeState_IDLE)
				}
			case <-ctx.Done():
				es.log.Info("Epoch service propcess epoch new exit")
				return
			}
		}
	}()
}

func (es *epochService) start(ctx context.Context) {
	es.processBlockAddedStart(ctx)
	es.processEpochNewStart(ctx)
}
