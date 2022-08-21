package configuration

import (
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type TestData struct {
	PrivKey        string
	InitDKGPrivKey string
}

var TestDatas = map[string]*TestData{
	"topia0": &TestData{PrivKey: "e8c059276e962c08799aa81a855df6a141a58cd0966b588309ca07ced02237ce0c89af103a079da1b95971d1425e490c0cb367c28285719eda47f3525f68e7bc", InitDKGPrivKey: ""},
	"topia1": &TestData{PrivKey: "2b5c9e866d10af7d6ea55156e5327a0c31460a3c02578b1e22148c2732346b234cfb276475c33ebb5f492026f99fe34dc72b2d904dfd6edc75231bd0cd61a82a", InitDKGPrivKey: ""},
	"topia2": &TestData{PrivKey: "9154a6138bdaeb94373bdde2c7211802d559d767fe83a039b78c5d500ca076c7aeb66f739745cdb5188988ce085413d4665dc377a1e8613abe65a3cdbf635c32", InitDKGPrivKey: ""},
	"topia3": &TestData{PrivKey: "a2441357796c99e6c3942beb6258120b34976e04935ce8a7afee15800d3738ab3f4aa03b9b6e4bc84ea9aa5c714bde038f57436dc09385538183164bc4ac446d", InitDKGPrivKey: "50b333e7537a26ed5cfe769926f0542836d79407793d8e54964e66dcaf60f3b1"},
	"topia4": &TestData{PrivKey: "6ecb91aa4e7b538bb1b05b4d1c780f0fdb91223d5b57b5e7f812db5a4a0caf0dfa70bdd25b759b65f5ea369d9642f2a6e26d4c12969f19d175f5cb3dfb2ebbcc", InitDKGPrivKey: "6d0c13ced273fe084580e75ca242c679c8bf55aa0f911112f2abf8851fd2086a"},
	"topia5": &TestData{PrivKey: "201f0989149fc5b73f2e749a3680ab5cc1607acc617ec6d6e32bcc117fff10dc6e0069494a04a8faa102f24fccb943ff121dec5a2a832cf839287b3e8137280b", InitDKGPrivKey: "25b164ed7eb12e0e5a52f29194b92466482f89783dcc467ee3a1a3ba42b6cc19"},
	"topia6": &TestData{PrivKey: "1fe2f1e645e5987feb4343469bc6c1bed5ef6aa93f65a0ce15599517a230e1186937e011596686ce7c0a48fe4ef0d91ead690fa8fe1ace9872010caeec6f0aba", InitDKGPrivKey: "30ef5edf691ea6aa126f7d581f0a8b491d337d8f4a48fa2a736845e4c6eda017"},
	"topia7": &TestData{PrivKey: "6d61237fa53c6627246b9c11776b56d7dabdea29acd47922be300950e55697f4a846b07b2d1f521c033ff890c4b6b8fd9c4fcbc177373e21697af90e4c8d67fc", InitDKGPrivKey: "7568f63e488f2b004d03ee3f9710f81aab0fb455751cf35166f0c83838e8ca3d"},
	"topia8": &TestData{PrivKey: "b017c91be5b12795304cb8394e5de327b554593e71a8bcc25c7b9f208adab48627a51d0efe3e0f67e62f76d01b8756362a71385747e784e8eb4407ab2e226751", InitDKGPrivKey: "25d557eafa7fbf9d6a23fe65e58ae282945ce28d793b970540db9fdb9ade1647"},
	"topia9": &TestData{PrivKey: "271db7767b5236d0ff4f6312ac0d21dfc7cda93edb12646ae1932e46e75b8b7b8c2bfacc917270bcff03b6cf40bbf22154ee36438e519f95f0892101f7120a89", InitDKGPrivKey: "2ebf6ecc8195567b8e7a68cb0774efce58002962ae9e80101bee05b5c087528d"},
}

type ConsensusConfiguration struct {
	EpochInterval            uint64 //the height number between two epochs
	DKGStartBeforeEpoch      uint64 //the starting height number of DKG before an epoch
	CrptyType                tpcrtypes.CryptType
	InitDKGPrivKey           string
	priKey                   tpcrtypes.PrivateKey
	ExecutionPrepareInterval time.Duration
	ProposerBlockMaxInterval time.Duration
	BlockMaxCyclePeriod      time.Duration
	MaxPrepareMsgCache       uint64
	BlocksPerEpoch           uint64
}

func DefConsensusConfiguration() *ConsensusConfiguration {
	return &ConsensusConfiguration{
		EpochInterval:            172800,
		DKGStartBeforeEpoch:      10,
		CrptyType:                tpcrtypes.CryptType_Ed25519,
		ExecutionPrepareInterval: 500 * time.Millisecond,
		ProposerBlockMaxInterval: 1000 * time.Millisecond,
		BlockMaxCyclePeriod:      5000 * time.Millisecond,
		MaxPrepareMsgCache:       50,
		BlocksPerEpoch:           50000,
	}
}
