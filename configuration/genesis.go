package configuration

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"strings"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

type GenesisData struct {
	ChainID          tpchaintypes.ChainID
	NetType          tpcmm.NetworkType
	Epoch            *tpcmm.EpochInfo
	Block            *tpchaintypes.Block
	BlockResult      *tpchaintypes.BlockResult
	GenesisNode      map[string]*tpcmm.NodeInfo //seed->nodeInfo
	GenesisExeDomain []*tpcmm.NodeDomainInfo
	SeedPeersMap     map[string][]*SeedPeer
}

func (genesis *GenesisData) Save(fileFullName string) error {
	dataBytes, err := json.Marshal(genesis)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fileFullName, dataBytes, fs.ModePerm)
}

func (genesis *GenesisData) Load() error {
	currentDir, _ := os.Getwd()
	lImdex := strings.LastIndex(currentDir, "topia")
	fileFullName := currentDir[:lImdex]
	fileFullName = path.Join(fileFullName, "topia", "configuration", "genesis.json")
	dataBytes, err := ioutil.ReadFile(fileFullName)
	if err != nil {
		return err
	}

	return json.Unmarshal(dataBytes, genesis)
}
