package configuration

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"io/fs"
	"io/ioutil"
)

type GenesisData struct {
	Epon        *chain.EpochInfo
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
