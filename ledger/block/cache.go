package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/bluele/gcache"

)

func (df *TopiaFile) FindBlockInCache(blockNum types.BlockNum) (*types.Block, error) {
	gc := gcache.New(20).
		LRU().
		Build()
	//gc.Set("key", "goodofthebest")
	value, err := gc.Get(blockNum)
	if err != nil {
		panic(err)
	}
	fmt.Println(value)

	return nil,err
}