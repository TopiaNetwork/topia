package block

import (
	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
	"os"
	"syscall"
)

func NewRollback(blocknum types.BlockNum) error{

}


func (Rfile *FileItem)AddRollback(blocknum types.BlockNum) error{
	fd, err := os.OpenFile(Rfile.File.Name(),os.O_RDWR|os.O_APPEND,0644)
	if err != nil{
		return  err
	}

	fd.Write(Uint64ToBytes(uint64(blocknum)))






}

func Removeblock(Datafile *FileItem,blocknum types.BlockNum)error{


}

func Removeindex(Indexfile *FileItem, blocknums []types.BlockNum)error {

	var alloffset uint64 = 0
	var startoffset uint64 = 0
	for _,blocknum := range blocknums{
		index, err := Indexfile.Findindex(blocknum)

		if err != nil {
			return err
		}
		alloffset = index.offset + alloffset
		startoffset = index.offset
	}

	filedata, err := os.OpenFile(Indexfile.File.Name(), os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}
	indexmmap, err := gommap.Map(filedata.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}

	



	return nil
}

func EmptyRollback(blocknum types.BlockNum)error{

}

