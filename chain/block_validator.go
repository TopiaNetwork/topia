package chain

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/Gurpartap/async"
	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/sign/bls"
	"github.com/hashicorp/go-multierror"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/consensus/vrf"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type blockValidator struct {
	log          tplog.Logger
	ledger       ledger.Ledger
	marshaler    codec.Marshaler
	cryptService tpcrt.CryptService
}

func (v *blockValidator) validateBase(curChainID tpchaintypes.ChainID, bh *tpchaintypes.BlockHead) error {
	if tpchaintypes.ChainID(bh.ChainID) != curChainID {
		return fmt.Errorf("Invalid chain ID: expected %s, actual %s", curChainID, tpchaintypes.ChainID(bh.ChainID))
	}

	if bh.Height > 1 && bh.ParentBlockHash == nil {
		return errors.New("Nil block ParentBlockHash")
	}

	if bh.Proposer == nil {
		return errors.New("Nil block proposer ")
	}

	if bh.Height > 1 && bh.Proof == nil {
		return errors.New("Nil block consensus proof ")
	}

	if bh.VRFProof == nil {
		return errors.New("Nil block vrf proof")
	}

	if bh.VRFProofHeight < bh.Height {
		return errors.New("Invalid vrf height")
	}

	if bh.VoteAggSignature == nil {
		return errors.New("Nil block vote aggregate signature")
	}

	return nil
}

func (v *blockValidator) proofVerify(compStateRN state.CompositionStateReadonly, bh *tpchaintypes.BlockHead) func() error {
	return func() error {
		if bh.Height == 1 {
			return nil
		}
		parentBlock, err := v.ledger.GetBlockStore().GetBlockByNumber(tpchaintypes.BlockNum(bh.Height - 1))
		if err != nil {
			return err
		}

		pHashBytes, _ := parentBlock.HashBytes()
		if !bytes.Equal(bh.ParentBlockHash, pHashBytes) {
			return errors.New("Invalid parent block reference")
		}

		csProof := tpchaintypes.ConsensusProof{
			ParentBlockHash: pHashBytes,
			Height:          parentBlock.Head.Height,
			AggSign:         parentBlock.Head.VoteAggSignature,
		}
		csProofBytes, err := v.marshaler.Marshal(csProof)
		if err != nil {
			v.log.Errorf("Marshal consensus proof failed: %v", err)
			return err
		}
		if !bytes.Equal(bh.Proof, csProofBytes) {
			return errors.New("Invalid consensus proof")
		}

		proposerSel := vrf.NewProposerSelector(vrf.ProposerSelectionType_Poiss, v.cryptService)

		hasher := tpcmm.NewBlake2bHasher(0)
		if err = binary.Write(hasher.Writer(), binary.BigEndian, bh.Epoch); err != nil {
			return err
		}
		if err = binary.Write(hasher.Writer(), binary.BigEndian, bh.VRFProofHeight); err != nil {
			return err
		}
		if _, err = hasher.Writer().Write(csProofBytes); err != nil {
			return err
		}
		proposerNode, err := compStateRN.GetNode(string(bh.Proposer))
		ok, err := proposerSel.VerifyVRF(tpcrtypes.Address(proposerNode.Address), hasher.Bytes(), bh.VRFProof)
		if err != nil {
			v.log.Errorf("Verify proposer selector vrf proof err: height %v", err)
			return err
		}
		if !ok {
			return fmt.Errorf("Invalid proposer vrf proof: height %d`", bh.Height)
		}

		totalActiveProposerWeight, err := compStateRN.GetActiveProposersTotalWeight()
		if err != nil {
			v.log.Errorf("Can't get total active proposer weight: %v", err)
			return err
		}
		proposerWeight := proposerNode.Weight

		winCount := proposerSel.SelectProposer(bh.VRFProof, big.NewInt(int64(proposerWeight)), big.NewInt(int64(totalActiveProposerWeight)))
		if winCount >= 1 {
			maxPri := proposerSel.MaxPriority(bh.VRFProof, winCount)
			if !bytes.Equal(bh.MaxPri, maxPri) {
				fmt.Errorf("Invalid block because of invalid proposer wining value: proposer %s, height %d", proposerNode.Address)
			}
		} else {
			return fmt.Errorf("Invalid block because of proposer not winner: proposer %s, height %d", proposerNode.Address)
		}

		return nil
	}
}

func (v *blockValidator) signVerify(compStateRN state.CompositionStateReadonly, bh *tpchaintypes.BlockHead) func() error {
	return func() error {
		var signInfo tpcrtypes.SignatureInfo
		err := json.Unmarshal(bh.VoteAggSignature, &signInfo)
		if err != nil {
			return nil
		}

		suite := bn256.NewSuiteG2()
		pubPolicy := suite.Point()
		err = pubPolicy.UnmarshalBinary(signInfo.PublicKey)
		if err != nil {
			return err
		}

		signBH, _ := bh.DeepCopy(bh)
		signBH.VoteAggSignature = nil
		signBHBytes, _ := v.marshaler.Marshal(&signBH)

		return bls.Verify(suite, pubPolicy, signBHBytes, signInfo.SignData)
	}
}

func (v *blockValidator) stateRootVerify(compStateRN state.CompositionStateReadonly, bh *tpchaintypes.BlockHead) func() error {
	return func() error {
		stateRoot, err := compStateRN.StateRoot()
		if err != nil {
			v.log.Errorf("Can't get state root: %v", err)
			return err
		}

		if !bytes.Equal(stateRoot, bh.StateRoot) {
			return fmt.Errorf("Invalid state root: height %d", bh.Height)
		}

		return nil
	}
}

func (v *blockValidator) txVerify(compStateRN state.CompositionStateReadonly, block *tpchaintypes.Block) func() error {
	return func() error {
		actualChunkCnt := uint32(len(block.Data.DataChunks))
		if block.Head.ChunkCount != actualChunkCnt {
			return fmt.Errorf("Invalid chunk count: expected %d, actual %d", block.Head.ChunkCount, actualChunkCnt)
		}

		if len(block.Head.HeadChunks) != len(block.Data.DataChunks) {
			return fmt.Errorf("Invalid chunk count: head count %d, data count %d", len(block.Head.HeadChunks), len(block.Data.DataChunks))
		}

		for i, dataChunkBytes := range block.Data.DataChunks {
			var headChunk tpchaintypes.BlockHeadChunk
			var dataChunk tpchaintypes.BlockDataChunk
			headChunk.Unmarshal(block.Head.HeadChunks[i])
			dataChunk.Unmarshal(dataChunkBytes)
			txRoot := txbasic.TxRootByBytes(dataChunk.Txs)
			if !bytes.Equal(txRoot, headChunk.TxRoot) {
				return fmt.Errorf("Invalid tx root: height %d", block.Head.Height)
			}
		}

		return nil
	}
}

func (v *blockValidator) processAwaitListResult(ctx context.Context, awaitList []async.ErrorFuture) error {
	var merr error
	for _, fut := range awaitList {
		if err := fut.AwaitContext(ctx); err != nil {
			merr = multierror.Append(merr, err)
		}
	}
	if merr != nil {
		mulErr := merr.(*multierror.Error)
		mulErr.ErrorFormat = func(es []error) string {
			if len(es) == 1 {
				return fmt.Sprintf("1 error occurred:\n\t* %+v\n\n", es[0])
			}

			points := make([]string, len(es))
			for i, err := range es {
				points[i] = fmt.Sprintf("* %+v", err)
			}

			return fmt.Sprintf(
				"%d errors occurred:\n\t%s\n\n",
				len(es), strings.Join(points, "\n\t"))
		}
		return mulErr
	}

	return merr
}

func (v *blockValidator) ValidateHeader(ctx context.Context, bh *tpchaintypes.BlockHead) error {
	compStateRN := state.CreateCompositionStateReadonlyAt(v.log, v.ledger, bh.Height-1)
	if compStateRN == nil {
		return fmt.Errorf("Can't get the responding composition state: height %d", bh.Height)
	}
	defer compStateRN.Stop()

	if err := v.validateBase(compStateRN.ChainID(), bh); err != nil {
		v.log.Errorf("Block head basic verify failed: %v", err)
		return err
	}

	awaitList := []async.ErrorFuture{
		async.Err(v.proofVerify(compStateRN, bh)),
		async.Err(v.stateRootVerify(compStateRN, bh)),
		async.Err(v.signVerify(compStateRN, bh)),
	}

	return v.processAwaitListResult(ctx, awaitList)
}

func (v *blockValidator) ValidateBlock(ctx context.Context, block *tpchaintypes.Block) error {
	compStateRN := state.CreateCompositionStateReadonlyAt(v.log, v.ledger, block.Head.Height-1)
	if compStateRN == nil {
		return fmt.Errorf("Can't get the responding composition state: height %d", block.Head.Height)
	}
	defer compStateRN.Stop()

	if err := v.validateBase(compStateRN.ChainID(), block.Head); err != nil {
		v.log.Errorf("Block head basic verify failed: %v", err)
		return err
	}

	awaitList := []async.ErrorFuture{
		async.Err(v.proofVerify(compStateRN, block.Head)),
		async.Err(v.stateRootVerify(compStateRN, block.Head)),
		async.Err(v.signVerify(compStateRN, block.Head)),
		async.Err(v.txVerify(compStateRN, block)),
	}

	return v.processAwaitListResult(ctx, awaitList)
}
