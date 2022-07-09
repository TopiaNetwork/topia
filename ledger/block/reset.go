package block

import (
	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
	"syscall"
)

func (df *FileItem) resetblock(block *types.Block) error {


	mmap, err := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer mmap.UnsafeUnmap()

	empty := make([]byte,df.Offset)
	copy(mmap[0:df.Offset],empty)
	_ = df.File.Sync()
	df.HeaderOffset = 0

	return  nil

}
