package consensus

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/lazyledger/smt"
	"math/big"

	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	ValidateTxMaxCount = 3
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

func (ev *executionResultValidate) resultValidateDataRequest(ctx context.Context, propMsg *ProposeMessage, validateTxCount int) (*ExeResultValidateRespMessage, [][]byte, [][]byte, error) {
	var randTxHashs [][]byte
	var randTxRSHashs [][]byte

	maxTxIndex := big.NewInt(int64(len(propMsg.TxHashs)))
	for i := 0; i < validateTxCount; i++ {
		randTxHashBig, err := rand.Int(rand.Reader, maxTxIndex)
		if err != nil {
			ev.log.Errorf("Generate validate tx hash index err: %d, %v", i, err)
			return nil, nil, nil, err
		}
		randTxHashIndex := randTxHashBig.Uint64()

		ev.log.Debugf("Get validate tx hash index: %d", randTxHashIndex)

		randTxHashs = append(randTxHashs, propMsg.TxHashs[randTxHashIndex])
		randTxRSHashs = append(randTxRSHashs, propMsg.TxResultHashs[randTxHashIndex])
	}

	validateReq := &ExeResultValidateReqMessage{
		ChainID:       propMsg.ChainID,
		Version:       propMsg.Version,
		Epoch:         propMsg.Epoch,
		Round:         propMsg.Round,
		Validator:     []byte(ev.nodeID),
		StateVersion:  propMsg.StateVersion,
		TxHashs:       randTxHashs,
		TxResultHashs: randTxRSHashs,
	}

	validateResp, err := ev.deliver.deliverResultValidateReqMessage(ctx, validateReq)

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

func (ev *executionResultValidate) Validate(ctx context.Context, propMsg *ProposeMessage) (string, bool, error) {
	if len(propMsg.TxHashs) == 0 && len(propMsg.TxResultHashs) == 0 {
		return "", true, nil
	}

	if len(propMsg.TxHashs) != len(propMsg.TxResultHashs) {
		err := fmt.Errorf("Propose message tx count %d not equal to tx result count %d", len(propMsg.TxHashs), len(propMsg.TxResultHashs))
		ev.log.Errorf("%v", err)
		return "", false, err
	}

	validateResp, txHashs, txResultHashs, err := ev.resultValidateDataRequest(ctx, propMsg, ValidateTxMaxCount)
	if err != nil {
		ev.log.Errorf("Deliver execution result validate req err: %v", err)
		return "", false, err
	}

	if len(validateResp.TxProofs) != ValidateTxMaxCount || len(validateResp.TxResultProofs) != ValidateTxMaxCount {
		err := fmt.Errorf("Invalid tx proof or tx result proof count: expected %d, %d; actual %d, %d", len(propMsg.TxHashs), len(propMsg.TxResultHashs), validateResp.TxProofs, validateResp.TxResultProofs)
		ev.log.Errorf("%v", err)
		return "", false, err
	}

	return ev.validateExeResultResp(propMsg.TxRoot(), propMsg.TxResultRoot(), txHashs, txResultHashs, validateResp)
}
