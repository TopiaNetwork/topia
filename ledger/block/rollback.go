package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	//"launchpad.net/gommap"
	"os"
	"strconv"
	//"strings"
	//"syscall"
)


type RollbackData struct {
	version uint32
	Startfromblock uint64
	datatime uint64

}
func NewRollback(blocknum types.BlockNum) (*FileItem, error){

	filepath := strconv.FormatInt(int64(blocknum), 10) + ".roll"

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, FILE_SIZE))
	if err != nil {
		return  nil,err
	}

	var tp  = FileItem{
		Filetype: RollbackFileType,
		File:   file,
		Offset: 0,
	}

	return &tp,nil
}


func (Rfile *FileItem)AddRollback(blocknum types.BlockNum) error{
	fd, err := os.OpenFile(Rfile.File.Name(),os.O_RDWR|os.O_APPEND,0644)
	if err != nil{
		return  err
	}

	fd.Write(Uint64ToBytes(uint64(blocknum)))


	return nil
}

func RemoveBlockhead(datafile *FileItem,offset uint64)error{
	StartBlock := GetStartblockFromFilename(datafile)
	n, err := strconv.ParseUint(StartBlock, 10, 64)
	if err != nil {
		return nil
	}
	fmt.Println(n)
	datammap := Getmmap(datafile.File.Name())

	size := offset - n
	buf := make([]byte, size)
	copy(datammap[offset:offset+size], buf)

	return nil
}


func RemoveBlockdata(datafile *FileItem,offset uint64)error{
	file := GetDataFilename(datafile)
	size := GetSize(file,offset)

	datammap := Getmmap(file)
	buf := make([]byte, size)
	copy(datammap[offset:size], buf)

	return nil
}

//func Removeindex(Indexfile *FileItem, blocknums []types.BlockNum)error {
func Removeindex(Indexfile *FileItem, blocknums []uint64) error {
	var alloffset uint64 = 0
	var startoffset uint64 = 0
	for _,blocknum := range blocknums{
		index, err := Indexfile.Findindex(types.BlockNum(blocknum))

		if err != nil {
			return err
		}
		alloffset = index.offset + alloffset
		startoffset = index.offset

		RemoveBlockdata(Indexfile,index.offset)
	}

	indexmmap := Getmmap(Indexfile.File.Name())
	buf := make([]byte, alloffset)
	copy(indexmmap[startoffset:startoffset+alloffset],buf)

	return nil
}

func (RollFile *FileItem)EmptyRollback(blocknum types.BlockNum)error{

	rollmmap := Getmmap(RollFile.File.Name())
	buf := make([]byte, RollFile.Offset)
	copy(rollmmap[0:RollFile.Offset],buf)

	return nil

}
