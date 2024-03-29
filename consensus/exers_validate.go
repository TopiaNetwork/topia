package consensus

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/lazyledger/smt"

	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	ValidateTxMinCount = 3
)

type executionResultValidate struct {
	log     tplog.Logger
	nodeID  string
	deliver messageDeliverI
}

func newExecutionResultValidate(log tplog.Logger, nodeID string, deliver messageDeliverI) *executionResultValidate {
	return &executionResultValidate{
		log:     log,
		nodeID:  nodeID,
		deliver: deliver,
	}
}

func (ev *executionResultValidate) resultValidateDataRequest(ctx context.Context, propMsg *ProposeMessage, propData *PropData, validateTxCount int) (*ExeResultValidateRespMessage, [][]byte, [][]byte, error) {
	var randTxHashs [][]byte
	var randTxRSHashs [][]byte

	maxTxIndex := big.NewInt(int64(len(propData.TxHashs)))
	for i := 0; i < validateTxCount; i++ {
		randTxHashBig, err := rand.Int(rand.Reader, maxTxIndex)
		if err != nil {
			ev.log.Errorf("Generate validate tx hash index err: %d, %v", i, err)
			return nil, nil, nil, err
		}
		randTxHashIndex := randTxHashBig.Uint64()

		ev.log.Debugf("Get validate tx hash index: %d", randTxHashIndex)

		randTxHashs = append(randTxHashs, propData.TxHashs[randTxHashIndex])
		randTxRSHashs = append(randTxRSHashs, propData.TxResultHashs[randTxHashIndex])
	}

	validateReq := &ExeResultValidateReqMessage{
		ChainID:       propMsg.ChainID,
		Version:       propData.Version,
		Epoch:         propMsg.Epoch,
		Round:         propMsg.Round,
		Validator:     []byte(ev.nodeID),
		StateVersion:  propMsg.StateVersion,
		TxHashs:       randTxHashs,
		TxResultHashs: randTxRSHashs,
	}

	validateResp, err := ev.deliver.deliverResultValidateReqMessage(ctx, string(propData.DomainID), validateReq)

	ev.log.Infof("Successfully send result validate req: state version %d, validator %s, self node %s", validateReq.StateVersion, ev.nodeID, ev.nodeID)

	return validateResp, randTxHashs, randTxRSHashs, err
}

func (ev *executionResultValidate) validateExeResultResp(origTxRoot []byte, origTxRSRoot []byte, txHashs [][]byte, txResultHashs [][]byte, validateResp *ExeResultValidateRespMessage) (string, bool, error) {
	deepSMSTTx := smt.NewDeepSparseMerkleSubTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New(), origTxRoot)
	deepSMSTTxRS := smt.NewDeepSparseMerkleSubTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New(), origTxRSRoot)

	for i := 0; i < len(validateResp.TxProofs); i++ {
		var txProofInfo smt.SparseMerkleProof
		var txRSProofInfo smt.SparseMerkleProof

		err := json.Unmarshal(validateResp.TxProofs[i], &txProofInfo)
		if err != nil {
			ev.log.Errorf("Invalid tx proof %d from executor %s", i, string(validateResp.Executor))
			return "", false, err
		}
		err = deepSMSTTx.AddBranch(txProofInfo, txHashs[i], txHashs[i])
		if err != nil {
			ev.log.Errorf("Invalid tx proof %d from executor %s, so add branch err: %v", i, string(validateResp.Executor), err)
			return "", false, err
		}

		err = json.Unmarshal(validateResp.TxResultProofs[i], &txRSProofInfo)
		if err != nil {
			ev.log.Errorf("Invalid tx result proof %d from executor %s", i, string(validateResp.Executor))
			return "", false, err
		}
		err = deepSMSTTxRS.AddBranch(txRSProofInfo, txResultHashs[i], txResultHashs[i])
		if err != nil {
			ev.log.Errorf("Invalid tx result proof %d from executor %s, so add branch err: %v", i, string(validateResp.Executor), err)
			return "", false, err
		}
	}

	if bytes.Compare(origTxRoot, deepSMSTTx.Root()) != 0 {
		err := fmt.Errorf("Invalid tx proof from executor %s", string(validateResp.Executor))
		ev.log.Errorf("%v", err)

		return "", false, err
	}

	if bytes.Compare(origTxRSRoot, deepSMSTTxRS.Root()) != 0 {
		err := fmt.Errorf("Invalid tx result proof from executor %s", string(validateResp.Executor))
		ev.log.Errorf("%v", err)

		return "", false, err
	}

	return string(validateResp.Executor), true, nil
}

func (ev *executionResultValidate) Validate(ctx context.Context, propMsg *ProposeMessage) (bool, error) {
	if len(propMsg.PropDatas) == 0 {
		return true, nil
	}

	var wg sync.WaitGroup
	oks := make([]bool, len(propMsg.PropDatas))
	errs := make([]error, len(propMsg.PropDatas))
	for i, _ := range propMsg.PropDatas {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			var propData PropData

			err := propData.Unmarshal(propMsg.PropDatas[i])
			if err != nil {
				err := fmt.Errorf("Unmarshal propDataBytes failed: %v", err)
				ev.log.Errorf("%v", err)
				oks[i] = false
				errs[i] = err
				return
			}

			if len(propData.TxHashs) != len(propData.TxResultHashs) {
				err := fmt.Errorf("Propose message tx count %d not equal to tx result count %d", len(propData.TxHashs), len(propData.TxResultHashs))
				ev.log.Errorf("%v", err)
				oks[i] = false
				errs[i] = err
				return
			}

			validateTxCount := ValidateTxMinCount
			if len(propData.TxHashs) < ValidateTxMinCount {
				validateTxCount = len(propData.TxHashs)
			}

			validateResp, txHashs, txResultHashs, err := ev.resultValidateDataRequest(ctx, propMsg, &propData, validateTxCount)
			if err != nil {
				ev.log.Errorf("Deliver execution result validate req err: %v", err)
				oks[i] = false
				errs[i] = err
				return
			}

			if len(validateResp.TxProofs) != validateTxCount || len(validateResp.TxResultProofs) != validateTxCount {
				err := fmt.Errorf("Invalid tx proof or tx result proof count: expected %d, %d; actual %d, %d", len(propData.TxHashs), len(propData.TxResultHashs), validateResp.TxProofs, validateResp.TxResultProofs)
				ev.log.Errorf("%v", err)
				oks[i] = false
				errs[i] = err
				return
			}

			executor, ok, err := ev.validateExeResultResp(propData.TxRoot(), propData.TxResultRoot(), txHashs, txResultHashs, validateResp)
			if !ok {
				ev.log.Errorf("Validate err by executor %s: %v", executor, err)
				oks[i] = false
				errs[i] = err
				return
			}

			oks[i] = true
			errs[i] = nil
		}(i)
	}
	wg.Wait()

	if len(oks) != len(errs) {
		err := fmt.Errorf("Must have other errors: oks'len %d, errs'len %d", len(oks), len(errs))
		ev.log.Errorf("%v", err)
		return false, err
	}

	var errM *multierror.Error
	for _, err := range errs {
		if err != nil {
			errM = multierror.Append(errM, err)
		}
	}

	if errM == nil {
		return true, nil
	}

	return false, errM
}
