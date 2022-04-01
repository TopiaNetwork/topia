package configuration

import (
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type ConsensusConfiguration struct {
	RoundDuration            time.Duration
	EpochInterval            uint64 //the round number between two epochs
	CrptyType                tpcrtypes.CryptType
	InitDKGPrivKey           string
	InitDKGPartPubKeys       []string
	ExecutionPrepareInterval time.Duration
}

func DefConsensusConfiguration() *ConsensusConfiguration {
	return &ConsensusConfiguration{
		RoundDuration:            500 * time.Millisecond,
		EpochInterval:            172800,
		CrptyType:                tpcrtypes.CryptType_Ed25519,
		ExecutionPrepareInterval: 500 * time.Millisecond,
	}
}
