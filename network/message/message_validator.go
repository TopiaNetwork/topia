package message

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type PubSubMessageValidator func(ctx context.Context, from peer.ID, msg *NetworkPubSubMessage) pubsub.ValidationResult

func TopicValidator(validators ...PubSubMessageValidator) pubsub.ValidatorEx {
	return func(ctx context.Context, fromPeer peer.ID, rawMsg *pubsub.Message) pubsub.ValidationResult {
		subMsg, ok := rawMsg.ValidatorData.(*NetworkPubSubMessage)
		if !ok {
			return pubsub.ValidationReject
		}

		result := pubsub.ValidationAccept
		for _, validator := range validators {
			switch res := validator(ctx, fromPeer, subMsg); res {
			case pubsub.ValidationReject:
				return res
			case pubsub.ValidationIgnore:
				result = res
			}
		}

		return result
	}
}
