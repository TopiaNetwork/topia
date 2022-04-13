package consensus

import (
	"context"
	"fmt"
	"github.com/TopiaNetwork/topia/ledger/backend"
	"log"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TopiaNetwork/kyber/v3"
	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	dkg "github.com/TopiaNetwork/kyber/v3/share/dkg/pedersen"
	vss "github.com/TopiaNetwork/kyber/v3/share/vss/pedersen"
	"github.com/TopiaNetwork/kyber/v3/util/encoding"
	"github.com/TopiaNetwork/kyber/v3/util/key"

	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

var partPubKeyChMap = make(map[string]chan *DKGPartPubKeyMessage)
var dealMsgChMap = make(map[string]chan *DKGDealMessage)
var dealRespMsgChMap = make(map[string]chan *DKGDealRespMessage)

var dkgExChangeMap = make(map[int]*dkgExchange)

var initPrivKeys []string
var initPubKeys []string
var initPrivKeysMap = make(map[string]string)
var initPubKeysMap = make(map[string]string)

func createTestDKGExChange(log tplog.Logger, nodeID string, ledger ledger.Ledger) *dkgExchange {
	deliver := &messageDeliverMock{
		log:              log,
		nodeID:           nodeID,
		initPubKeys:      initPubKeys,
		partPubKeyChMap:  partPubKeyChMap,
		dealMsgChMap:     dealMsgChMap,
		dealRespMsgChMap: dealRespMsgChMap,
	}

	dkgEx := newDKGExchange(log, "testdkg", nodeID, partPubKeyChMap[nodeID], dealMsgChMap[nodeID], dealRespMsgChMap[nodeID], initPrivKeysMap[nodeID], deliver, ledger)
	dkgEx.nodeID = nodeID
	dkgEx.dkgExData.initDKGPartPubKeys.Store(initPubKeysMap)

	return dkgEx
}

func creatKeyPairs(suite *bn256.Suite, nParticipant int) {
	initPrivKeys = make([]string, nParticipant)
	initPubKeys = make([]string, nParticipant)
	for i := 0; i < nParticipant; i++ {
		nodei := fmt.Sprintf("node%d", i)
		keyPair := key.NewKeyPair(suite)
		initPrivKeys[i], _ = encoding.ScalarToStringHex(suite, keyPair.Private)
		initPubKeys[i], _ = encoding.PointToStringHex(suite, keyPair.Public)
		initPrivKeysMap[nodei] = initPrivKeys[i]
		initPubKeysMap[nodei] = initPubKeys[i]
	}
}

func createTestDKGNode(log tplog.Logger, nParticipant int) {
	lg := ledger.NewLedger(".", "dkgtest", log, backend.BackendType_Badger)

	suite := bn256.NewSuiteG2()
	creatKeyPairs(suite, nParticipant)
	for i := 0; i < nParticipant; i++ {
		nodei := fmt.Sprintf("node%d", i)
		partPubKeyChMap[nodei] = make(chan *DKGPartPubKeyMessage, PartPubKeyChannel_Size)
		dealMsgChMap[nodei] = make(chan *DKGDealMessage, DealMSGChannel_Size)
		dealRespMsgChMap[nodei] = make(chan *DKGDealRespMessage, DealRespMsgChannel_Size)

		dkgEx := createTestDKGExChange(log, nodei, lg)
		dkgEx.startLoop(context.Background())

		//dkgCrypt := newDKGCrypt(log, 10 /*suite, */, initPrivKeys[i], initPubKeys, 2*nParticipant/3+1, nParticipant)
		//dkgEx.setDKGCrypt(dkgCrypt)

		dkgExChangeMap[i] = dkgEx
	}
}

func TestDkgExchangeLoop(t *testing.T) {
	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")

	nParticipant := 3

	createTestDKGNode(log, nParticipant)

	var wg sync.WaitGroup
	msg := []byte("test dkg")
	signData := make([][]byte, 0)

	for i := 0; i < nParticipant; i++ {
		dkgExChangeMap[i].initWhenStart(10)

		for t, verfer := range dkgExChangeMap[i].dkgCrypt.dkGenerator.Verifiers() {
			longterm, pub := verfer.Key()
			log.Infof("After init, Verifier t %d: longterm=%s, pub=%s, index=%d", t, longterm.String(), pub.String(), verfer.Index())
		}
	}

	for i := 0; i < nParticipant; i++ {
		wg.Add(1)
		dkgExChangeMap[i].start(10)
		go func(dkgEx *dkgExchange, index int) {
			defer wg.Done()
			for {
				select {
				case isFinished := <-dkgEx.finishedCh:
					if isFinished {
						log.Infof("%d DKG finished", index)
						require.Equal(t, nParticipant, len(dkgEx.dkgCrypt.dkGenerator.QualifiedShares()))
						require.Equal(t, nParticipant, len(dkgEx.dkgCrypt.dkGenerator.QUAL()))
						log.Infof("qualified shares: %v", dkgEx.dkgCrypt.dkGenerator.QualifiedShares())
						log.Infof("QUAL: %v", dkgEx.dkgCrypt.dkGenerator.QUAL())

						signDataT, _, err := dkgEx.dkgCrypt.Sign(msg)
						require.Equal(t, nil, err)
						//err = dkgEx.dkgCrypt.Verify(msg, signDataT)
						//require.Equal(t, nil, err)

						signData = append(signData, signDataT)

						dkgEx.stop()
						return
					}
				}
			}
		}(dkgExChangeMap[i], i)
	}

	wg.Wait()

	/*
		aggPubKey := dkgExChangeMap[0].dkgCrypt.AggregatePublicKeys()
		require.NotEqual(t, nil, aggPubKey)
		sds, err1 := dkgExChangeMap[0].dkgCrypt.AggregateSignatures(signData...)
		require.Equal(t, nil, err1)
		dkgExChangeMap[0].dkgCrypt.VerifyAggSignatures(aggPubKey, msg, sds)
	*/

	sig, err := dkgExChangeMap[0].dkgCrypt.RecoverSig(msg, signData)
	require.Equal(t, nil, err)
	err = dkgExChangeMap[0].dkgCrypt.Verify(msg, sig)
	require.Equal(t, nil, err)

	log.Info("All dkg finished")
}

func TestDkgExchangeBase(t *testing.T) {
	n := 3
	suite := bn256.NewSuiteG2()
	creatKeyPairs(suite, n)

	initPartPubKeyP := make([]kyber.Point, n)
	for i := 0; i < len(initPubKeys); i++ {
		partPubKey, err := encoding.StringHexToPoint(suite, initPubKeys[i])
		if err != nil {
			log.Panicf("Invalid initPubKey %s", initPubKeys[i])
		}
		initPartPubKeyP[i] = partPubKey
	}

	dkgGenArray := make([]*dkg.DistKeyGenerator, n)
	for i := 0; i < n; i++ {
		priKey, _ := encoding.StringHexToScalar(suite, initPrivKeys[i])
		dkgGenArray[i], _ = dkg.NewDistKeyGenerator(suite, priKey, initPartPubKeyP, 2*n/3+1)
	}

	dealMap := make(map[uint32][]*dkg.Deal)
	for i := 0; i < n; i++ {
		deals, _ := dkgGenArray[i].Deals()
		for m, deal := range deals {
			t.Logf("i=%d, m=%d, index=%d\n", i, m, deal.Index)
			dealMap[uint32(m)] = append(dealMap[uint32(m)], deal)
			/*fmt.Printf("m=%d, index=%d\n", m, deal.Index)
			_, err := dkgGenArray[1].ProcessDeal(deal)
			if err != nil {
				fmt.Printf("m=%d, index=%d 1 fail\n", m, deal.Index)
			} else {
				fmt.Printf("m=%d, index=%d 1 succ\n", m, deal.Index)
			}
			assert.Equal(t, nil, err)

			_, err = dkgGenArray[2].ProcessDeal(deal)
			if err != nil {
				fmt.Printf("m=%d, index=%d 2 fail\n", m, deal.Index)
			} else {
				fmt.Printf("m=%d, index=%d 2 succ\n", m, deal.Index)
			}
			assert.Equal(t, nil, err)
			*/
		}
	}

	dealRespMap := make(map[uint32][]*dkg.Response)
	for i := 0; i < n; i++ {
		for _, deal := range dealMap[uint32(i)] {
			resp, err := dkgGenArray[i].ProcessDeal(deal)
			t.Logf("i=%d, resp Index=%d", i, resp.Index)
			assert.Equal(t, nil, err)
			assert.Equal(t, vss.StatusApproval, resp.Response.Status)
			for j := 0; j < n; j++ {
				if j == i {
					continue
				}
				dealRespMap[uint32(j)] = append(dealRespMap[uint32(j)], resp)
			}
		}
	}

	for i := 0; i < n; i++ {
		for _, resp := range dealRespMap[uint32(i)] {
			_, err := dkgGenArray[i].ProcessResponse(resp)
			assert.Equal(t, nil, err)
		}
		t.Logf("DKG %d verifier response len: i, %d", i, len(dkgGenArray[i].Verifiers()[0].Responses()))
		assert.Equal(t, true, dkgGenArray[i].Certified())
	}
}
