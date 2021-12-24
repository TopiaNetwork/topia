package p2p

import (
	"context"
	"fmt"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network/message"
)

type P2PPubSubService struct {
	sync.Mutex
	ctx             context.Context
	log             tplog.Logger
	topicValidation bool
	pubSub          *pubsub.PubSub
	p2pService      *P2PService
	topics          map[string]*pubsub.Topic
	subs            map[string]*pubsub.Subscription
}

func NewP2PPubSubService(ctx context.Context, log tplog.Logger, topicValidation bool, pubSub *pubsub.PubSub, p2pService *P2PService) *P2PPubSubService {
	return &P2PPubSubService{
		ctx:             ctx,
		log:             tplog.CreateModuleLogger(logcomm.InfoLevel, "P2PPubSubService", log),
		topicValidation: topicValidation,
		pubSub:          pubSub,
		p2pService:      p2pService,
		topics:          make(map[string]*pubsub.Topic),
		subs:            make(map[string]*pubsub.Subscription),
	}
}

func (ps *P2PPubSubService) Subscribe(ctx context.Context, topic string, validators ...message.PubSubMessageValidator) error {
	ps.Lock()
	defer ps.Unlock()

	ps.pubSub.GetTopics()
	tp, found := ps.topics[topic]
	var err error
	if !found {
		if ps.topicValidation {
			topicValidator := message.TopicValidator(validators...)
			if err := ps.pubSub.RegisterTopicValidator(
				topic, topicValidator, pubsub.WithValidatorInline(true),
			); err != nil {
				return fmt.Errorf("failed to register topic validator: %w", err)
			}
		}

		tp, err = ps.pubSub.Join(topic)
		if err != nil {
			if ps.topicValidation {
				if err := ps.pubSub.UnregisterTopicValidator(topic); err != nil {
				}
			}

			return fmt.Errorf("could not join topic (%s): %w", topic, err)
		}

		ps.topics[topic] = tp
	}

	s, err := tp.Subscribe()
	if err != nil {
		return fmt.Errorf("could not subscribe to topic (%s): %w", topic, err)
	}

	ps.subs[topic] = s

	go func(ctxPUB context.Context, subscr *pubsub.Subscription) {
		for {
			psMsg, err := subscr.Next(ctxPUB)
			if err != nil {
				ps.log.Warnf("error from message subscription: ", err)
				if ctxPUB.Err() != nil {
					ps.log.Warn("quitting HandleIncomingMessages loop")
					return
				}
				continue
			}

			if pubMsg, ok := psMsg.ValidatorData.(*message.NetworkPubSubMessage); ok {
				err := ps.p2pService.dispatch(pubMsg.ModuleName, pubMsg)
				if err != nil {
					ps.log.Errorf("can't dispatch the pubsub message from peerID=%s", pubMsg.FromPeerID)
				}
			} else {
				ps.log.Errorf("invalid pubsub message from peerID=%s", pubMsg.FromPeerID)
			}
		}
	}(ctx, s)

	return err
}

func (ps *P2PPubSubService) UnSubscribe(topic string) error {
	ps.Lock()
	defer ps.Unlock()
	// Remove the Subscriber from the cache
	if s, found := ps.subs[topic]; found {
		s.Cancel()
		ps.subs[topic] = nil
		delete(ps.subs, topic)
	}

	tp, found := ps.topics[topic]
	if !found {
		err := fmt.Errorf("could not find topic (%s)", topic)
		return err
	}

	if ps.topicValidation {
		if err := ps.pubSub.UnregisterTopicValidator(topic); err != nil {
		}
	}

	err := tp.Close()
	if err != nil {
		err = fmt.Errorf("could not close topic (%s): %w", topic, err)
		return err
	}
	ps.topics[topic] = nil
	delete(ps.topics, topic)

	return err
}

func (ps *P2PPubSubService) Publish(ctx context.Context, topic string, data []byte) error {
	p2pTopic, found := ps.topics[topic]
	if !found {
		return fmt.Errorf("could not find topic (%s)", topic)
	}
	err := p2pTopic.Publish(ctx, data)
	if err != nil {
		return fmt.Errorf("could not publish top topic (%s): %w", topic, err)
	}
	return nil
}
