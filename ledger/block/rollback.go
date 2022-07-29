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
func NewRollback(blocknum types.BlockNum) (FileItem, error){

	filepath := strconv.FormatInt(int64(blocknum), 10) + ".roll"

	file, _ := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, FILE_SIZE))
	//if err != nil {
	//	return  nil,err
	//}

	var tp  = FileItem{
		Filetype: RollbackFileType,
		File:   file,
		Offset: 0,
	}

	return tp,nil
}


func (Rfile *FileItem)AddRollback(blocknum types.BlockNum) error{
	fd, err := os.OpenFile(Rfile.File.Name(),os.O_RDWR|os.O_APPEND,0644)
	if err != nil{
		return  err
	}
	indxfilename := GetIndexFilename(Rfile)
	StartBlock := GetStartblockFromFilenamestring(indxfilename)
	n, err := strconv.ParseUint(StartBlock, 10, 64)
	if err != nil {
		return nil
	}

	indexfile := getIndexfile(strconv.FormatUint(n, 10))
	// add latestblock
	fd.Write(Uint64ToBytes(n + indexfile.Offset))


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
	start := size*uint64(HEADER_SLICE_SIZE)
	buf := make([]byte, FILE_HEADER_SIZE-start)

	copy(datammap[start:FILE_HEADER_SIZE], buf)

	return nil
}


func RemoveBlockdata(datafile *FileItem,offset uint64)error{
	file := GetDataFilename(datafile)
	//size := GetSize(file,offset)
	StartBlock := GetStartblockFromFilenamestring(file)
	n, err := strconv.ParseUint(StartBlock, 10, 64)
	if err != nil {
		return nil
	}
	Nowdatafile := getDatafile(strconv.FormatUint(n, 10))
	datammap := Getmmap(file)
	buf := make([]byte, Nowdatafile.Offset-offset+10000)// ???? empty all the block
	copy(datammap[offset:Nowdatafile.Offset+10000], buf)

	return nil
}

//func Removeindex(Indexfile *FileItem, blocknums []types.BlockNum)error {
func RemoveBlock(Indexfile *FileItem,blocknum types.BlockNum) error {

	//var startoffset uint64 = 0

	index, err := Indexfile.Findindex(blocknum)

	if err != nil {
		return err
	}

	//startoffset = index.position

	RemoveBlockdata(Indexfile,index.position)

	StartBlock := GetStartblockFromFilenamestring(Indexfile.File.Name())
	n, err := strconv.ParseUint(StartBlock, 10, 64)
	if err != nil {
		return nil
	}
	indexmmap := Getmmap(Indexfile.File.Name())
	start := (uint64(blocknum)-n)*INDEX_SLICE_SIZE
	indexlen := Indexfile.Offset - start + INDEX_SLICE_SIZE
	buf := make([]byte, indexlen)
	copy(indexmmap[start:start + indexlen],buf)

	return nil
}

func (RollFile *FileItem)EmptyRollback(blocknum types.BlockNum)error{

	rollmmap := Getmmap(RollFile.File.Name())
	buf := make([]byte, RollFile.Offset)
	copy(rollmmap[0:RollFile.Offset],buf)

	return nil

}

