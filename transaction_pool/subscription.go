package transactionpool

import (
"context"
	"fmt"
	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network/message"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"sync"
)

type TxPoolPubSubService struct {
	sync.Mutex
	ctx             	context.Context
	log             	tplog.Logger
	topicValidation 	bool
	pubSub         	 	*pubsub.PubSub
	txPoolService    	*transactionPool
	topics          	map[string]*pubsub.Topic
	subs            	map[string]*pubsub.Subscription
	err          		chan error

}

func NewTxPoolPubSubService(ctx context.Context, log tplog.Logger, topicValidation bool, pubSub *pubsub.PubSub, txPool *transactionPool) *TxPoolPubSubService {
	return &TxPoolPubSubService{
		ctx:             ctx,
		log:             tplog.CreateModuleLogger(logcomm.InfoLevel, "P2PPubSubService", log),
		topicValidation: topicValidation,
		pubSub:          pubSub,
		txPoolService:   txPool,
		topics:          make(map[string]*pubsub.Topic),
		subs:            make(map[string]*pubsub.Subscription),
	}
}



func (ts *TxPoolPubSubService) Subscribe(ctx context.Context, topic string, validators ...message.PubSubMessageValidator) error {
	ts.Lock()
	defer ts.Unlock()

	ts.pubSub.GetTopics()
	tp, found := ts.topics[topic]
	var err error
	if !found {
		if ts.topicValidation {
			topicValidator := message.TopicValidator(validators...)
			if err := ts.pubSub.RegisterTopicValidator(
				topic, topicValidator, pubsub.WithValidatorInline(true),
			); err != nil {
				return fmt.Errorf("failed to register topic validator: %w", err)
			}
		}

		tp, err = ts.pubSub.Join(topic)
		if err != nil {
			if ts.topicValidation {
				if err := ts.pubSub.UnregisterTopicValidator(topic); err != nil {
				}
			}

			return fmt.Errorf("could not join topic (%s): %w", topic, err)
		}

		ts.topics[topic] = tp
	}

	s, err := tp.Subscribe()
	if err != nil {
		return fmt.Errorf("could not subscribe to topic (%s): %w", topic, err)
	}

	ts.subs[topic] = s

	go func(ctxPUB context.Context, subscr *pubsub.Subscription) {
		for {
			psMsg, err := subscr.Next(ctxPUB)
			if err != nil {
				ts.log.Warnf("error from message subscription: ", err)
				if ctxPUB.Err() != nil {
					ts.log.Warn("quitting HandleIncomingMessages loop")
					return
				}
				continue
			}
			pubMsg := psMsg.ValidatorData.(*message.NetworkPubSubMessage)
			ts.txPoolService.Dispatch(ctxPUB, pubMsg.Data)
		}
	}(ctx, s)

	return err
}

func (ts *TxPoolPubSubService) UnSubscribe(topic string) error {
	ts.Lock()
	defer ts.Unlock()
	// Remove the Subscriber from the cache
	if s, found := ts.subs[topic]; found {
		s.Cancel()
		ts.subs[topic] = nil
		delete(ts.subs, topic)
	}

	tp, found := ts.topics[topic]
	if !found {
		err := fmt.Errorf("could not find topic (%s)", topic)
		return err
	}

	if ts.topicValidation {
		if err := ts.pubSub.UnregisterTopicValidator(topic); err != nil {
		}
	}

	err := tp.Close()
	if err != nil {
		err = fmt.Errorf("could not close topic (%s): %w", topic, err)
		return err
	}
	ts.topics[topic] = nil
	delete(ts.topics, topic)

	return err
}

func (ts *TxPoolPubSubService) Publish(ctx context.Context, topic string, data []byte) error {
	txPoolTopic, found := ts.topics[topic]
	if !found {
		return fmt.Errorf("could not find topic (%s)", topic)
	}
	err := txPoolTopic.Publish(ctx, data)
	if err != nil {
		return fmt.Errorf("could not publish top topic (%s): %w", topic, err)
	}
	return nil
}

