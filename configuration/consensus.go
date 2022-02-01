package configuration

import (
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	"time"
)

type ConsensusConfiguration struct {
	RoundDuration      time.Duration
	EpochInterval      uint64 //the round number between two epochs
	CrptyType          tpcrt.CryptServiceType
	InitDKGPrivKey     string
	InitDKGPartPubKeys []string
}
