package configuration

import (
	"github.com/TopiaNetwork/topia/crypt/types"
	"time"
)

type ConsensusConfiguration struct {
	RoundDuration      time.Duration
	EpochInterval      uint64 //the round number between two epochs
	CrptyType          types.CryptType
	InitDKGPrivKey     string
	InitDKGPartPubKeys []string
}
