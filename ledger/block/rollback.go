package block

import (
	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
	"os"
	"strings"
	"syscall"
)


type RollbackData struct {
	version uint32
	Startfromblock uint64
	datatime uint64

}
func NewRollback(blocknum types.BlockNum) error{

}


func (Rfile *FileItem)AddRollback(blocknum types.BlockNum) error{
	fd, err := os.OpenFile(Rfile.File.Name(),os.O_RDWR|os.O_APPEND,0644)
	if err != nil{
		return  err
	}

	fd.Write(Uint64ToBytes(uint64(blocknum)))






}

func RemoveBlockdata(Datafile *FileItem,blocknum types.BlockNum)error{

	//find the data file and remove


	copy(mmap[df.Offset:df.Offset+4],versionbyte)





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

		RemoveBlockdata(&Indexfile.File.Name(),index.offset)
	}

	filedata, err := os.OpenFile(Indexfile.File.Name(), os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}
	indexmmap, err := gommap.Map(filedata.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}

	copy()



	return nil
}

func EmptyRollback(blocknum types.BlockNum)error{

}

