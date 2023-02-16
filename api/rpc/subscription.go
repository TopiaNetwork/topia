package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/eventhub"
	mapset "github.com/deckarep/golang-set"
	"strconv"
	"sync"
)

var errExceedSubsLimit = errors.New("your client's subscription to the event exceed maximum limit")

// subscriptionCenter manage all subs of a server
type subscriptionCenter struct {
	ProxyObj struct {
		Subscribe      func(remoteHost string, eventName string, filterString string) (subID int, err error) `grant:"read"`
		UnSubscribe    func(remoteHost string, eventName string, subID int) (err error)                      `grant:"read"`
		UnSubscribeAll func(remoteHost string) (err error)                                                   `grant:"read"`
	}
	eventSubscribers map[string]mapset.Set                       // eventName -> subscriber
	subscriptions    map[subscription]subscriptionSendController // all subscription
	wsServers        *sync.Map                                   // remoteAddr -> *WebsocketServer for subs msg sending
	mu               sync.RWMutex
}

type subscriber struct {
	host  string // ip:port specify client
	subID int    // subscription ID from 1 to ClientMaxSubsToOneEvent
}

type subscription struct {
	remoteHost   string
	eventName    string
	subID        int
	filterString string
}

type subscriptionSendController struct {
	cancel context.CancelFunc // for cancelling goroutine that get data from queue and send it
	queue  *Queue
}

func newSubCenter(wsServers *sync.Map) *subscriptionCenter {
	subCenter := &subscriptionCenter{
		eventSubscribers: make(map[string]mapset.Set),
		subscriptions:    make(map[subscription]subscriptionSendController),
		wsServers:        wsServers,
	}
	subCenter.init()
	return subCenter

}

func (sc *subscriptionCenter) Subscribe(remoteHost string, eventName string, filterString string) (subID int, err error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if !sc.isLegalEventName(eventName) {
		return 0, errors.New("no such event")
	}

	var addSuccess = false
	for i := 1; i <= ClientMaxSubsToOneEvent; i++ {
		addSuccess = sc.eventSubscribers[eventName].Add(
			subscriber{
				host:  remoteHost,
				subID: i,
			})
		if addSuccess {
			subID = i
			break
		}
	}
	if !addSuccess {
		return 0, errExceedSubsLimit
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	queueForSub := newQueue()

	sc.subscriptions[subscription{
		remoteHost:   remoteHost,
		eventName:    eventName,
		subID:        subID,
		filterString: filterString,
	}] = subscriptionSendController{
		cancel: cancelFunc,
		queue:  queueForSub,
	}

	// constantly get msg from queue and send it
	go func(c context.Context) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic:", r)
			}
		}()

		val, find := sc.wsServers.Load(remoteHost)
		if !find {
			fmt.Println("don't find ws server for ", remoteHost)
			return
		}
		websocketServer := val.(*WebsocketServer)

		for {

			select {
			case <-c.Done(): // cancel has been called by unsub
				return
			case eventMsg, ok := <-queueForSub.wait():
				if !ok {
					continue
				}
				// TODO msg go through filter
				eventMsgBytes, err := json.Marshal(eventMsg)
				if err != nil {
					fmt.Println(err.Error())
					queueForSub.removeHead()
					continue
				}

				message, err := EncodeMessage(MsgSubscribe, strconv.Itoa(subID), eventName, "", &ErrMsg{}, eventMsgBytes)
				if err != nil {
					fmt.Println(err.Error())
					queueForSub.removeHead()
					continue
				}

				websocketServer.send <- message

				queueForSub.removeHead()
			}
		}

	}(ctx)

	return subID, nil
}

func (sc *subscriptionCenter) UnSubscribe(remoteHost string, eventName string, subID int) (err error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if !sc.isLegalEventName(eventName) {
		return errors.New("no such event")
	}
	if subID <= 0 || subID > ClientMaxSubsToOneEvent {
		return errors.New("invalid subID")
	}

	suber := subscriber{
		host:  remoteHost,
		subID: subID,
	}
	if !sc.containsSubscription(suber, eventName) {
		return errors.New("no such subscription")
	}

	var subToRemove subscription
	for sub := range sc.subscriptions {
		if sub.remoteHost == remoteHost && sub.eventName == eventName && sub.subID == subID {
			subToRemove = sub
			break
		}
	}

	sc.subscriptions[subToRemove].queue.clear()
	sc.subscriptions[subToRemove].cancel() // cancel goroutine that get msg from queue and send it
	delete(sc.subscriptions, subToRemove)

	return nil
}

func (sc *subscriptionCenter) UnSubscribeAll(remoteHost string) (err error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	var subsToRemove []subscription
	for sub := range sc.subscriptions {
		if sub.remoteHost == remoteHost {
			subsToRemove = append(subsToRemove, sub)
		}
	}

	for _, sub := range subsToRemove {
		sc.eventSubscribers[sub.eventName].Remove(
			subscriber{
				host:  sub.remoteHost,
				subID: sub.subID,
			})

		sc.subscriptions[sub].queue.clear()
		sc.subscriptions[sub].cancel()
		delete(sc.subscriptions, sub)
	}

	return nil
}

func (sc *subscriptionCenter) receiveSubMsg(msg eventhub.EventMsg) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for sub := range sc.subscriptions {
		if sub.eventName == msg.Name {
			err := sc.subscriptions[sub].queue.enqueue(msg)
			if err != nil { // queue is full
				fmt.Println(err.Error())
				continue
			}
		}
	}
}

func (sc *subscriptionCenter) init() {
	sc.eventSubscribers[eventhub.EventName_TxPoolChanged] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_TxPrepared] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_TxRollbacked] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_TxCommited] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_BlockCreated] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_BlockCommited] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_BlockVerified] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_BlockConfirmed] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_BlockAdded] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_BlockRevert] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_EpochNew] = mapset.NewSet()
	sc.eventSubscribers[eventhub.EventName_ContractInvoked] = mapset.NewSet()
}

func (sc *subscriptionCenter) isLegalEventName(eventName string) bool {
	_, ok := sc.eventSubscribers[eventName]
	return ok
}

func (sc *subscriptionCenter) containsSubscription(s subscriber, eventName string) bool {
	return sc.eventSubscribers[eventName].Contains(s)
}
