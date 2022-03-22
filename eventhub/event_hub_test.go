package eventhub

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/stretchr/testify/assert"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tx "github.com/TopiaNetwork/topia/transaction"
)

func receivedTxCallBack_test(ctx context.Context, data interface{}) error {
	switch recvData := data.(type) {
	case *tx.Transaction:
		fmt.Println("Received tx")
		return nil
	default:
		return fmt.Errorf("Invalid type:%v", recvData)
	}
}

func TestEventHub(t *testing.T) {
	sysActor := actor.NewActorSystem()
	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	evHub := NewEventHub(tplogcmm.InfoLevel, testLog)

	err := evHub.Start(sysActor)
	defer evHub.Stop()

	assert.Equal(t, nil, err)

	evHub.Observe(context.Background(), EventName_TxReceived, receivedTxCallBack_test)
	evHub.Trig(context.Background(), EventName_TxReceived, &tx.Transaction{})

	time.Sleep(time.Second * 5)
}
