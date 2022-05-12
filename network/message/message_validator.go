package message

import (
	"context"
	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/transaction/basic"
	transactionpool "github.com/TopiaNetwork/topia/transaction_pool"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type ValidationResult int

const (
	ValidationAccept    = ValidationResult(0)
	ValidationReject    = ValidationResult(1)
	ValidationIgnore    = ValidationResult(2)
	validationThrottled = ValidationResult(-1)
)

type PubSubMessageValidator func(ctx context.Context, isLocal bool, data []byte) ValidationResult

func TopicValidator(localPeer peer.ID, log tplog.Logger, validators ...PubSubMessageValidator) pubsub.ValidatorEx {
	return func(ctx context.Context, remotePeer peer.ID, rawMsg *pubsub.Message) pubsub.ValidationResult {
		isLocal := false

		log.Debugf("Receive pubsub msg from %s", remotePeer.String())
		if remotePeer == localPeer {
			isLocal = true
		}

		var subMsg NetworkPubSubMessage
		marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
		err := marshaler.Unmarshal(rawMsg.Data, &subMsg)
		if err != nil {
			log.Errorf("Invalid sub message from %s", remotePeer.String())
			return pubsub.ValidationReject
		}

		result := pubsub.ValidationAccept
		for _, validator := range validators {
			switch res := validator(ctx, isLocal, subMsg.Data); res {
			case ValidationReject:
				return pubsub.ValidationResult(res)
			case ValidationIgnore:
				result = pubsub.ValidationResult(res)
			}
		}

		if result == pubsub.ValidationAccept {
			rawMsg.ValidatorData = &subMsg
		}

		return result
	}
}

func TxPoolMessageValidate(ctx context.Context, isLocal bool, data []byte) ValidationResult {
	msg := &transactionpool.TxMessage{}
	msg.Unmarshal(data)
	var tx basic.Transaction
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	err := marshaler.Unmarshal(msg.Data, &tx)
	if err != nil {
		return ValidationIgnore
	}
	if isLocal {
		return ValidationAccept
	}
	if types.ChainID(tx.Head.ChainID) != "" {
		return ValidationReject
	}
	return ValidationAccept
}
