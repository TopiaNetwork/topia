package configuration

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

type GenesisData struct {
	NetType     tpcmm.NetworkType
	Epon        *tpcmm.EpochInfo
	Block       *tpchaintypes.Block
	BlockResult *tpchaintypes.BlockResult
}

func (genesis *GenesisData) Save(fileFullName string) error {
	dataBytes, err := json.Marshal(genesis)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fileFullName, dataBytes, fs.ModePerm)
}

func (genesis *GenesisData) Load(fileFullName string) error {
	dataBytes, err := ioutil.ReadFile(fileFullName)
	if err != nil {
		return err
	}

	return json.Unmarshal(dataBytes, genesis)
}
