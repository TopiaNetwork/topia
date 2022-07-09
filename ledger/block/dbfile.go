package block

import (
	"fmt"
	"os"
	"strconv"


	"github.com/TopiaNetwork/topia/chain/types"
)


const FILE_SIZE = 10000 * 1000
const FILE_HEADER_SIZE = 10000

type FileType uint16

const (
	DataFileType       FileType = 0
	IndexFileType      FileType = 1
	DataHeaderType      FileType = 2
	BackFileType      FileType = 3
)

var FileNameOpening = ""
var Indexoffset = 0
//var Transoffset = 0

type FileItem struct {
	//header
	Filetype FileType //0,data;1,index;2,transactionindex
	File   *os.File
	Offset uint64
	HeaderOffset uint64
	Bloom *BloomFilter
	//HeaderSize uint64

}

type IndexFileItem struct {
	//header
	Filetype FileType //0,data;1,index;2,transactionindex
	File   *os.File
	Offset uint64
	HeaderOffset uint64
	//HeaderSize uint64

}

type DataItem struct{
	version uint32
	offset uint64
	size uint16
	crc uint64
	data *types.Block
}


type IndexItem struct{
	version uint16
	position uint64
	offset uint64
}


type TransIndex struct{
	Version uint16
	Txid int64
	BlockHeight uint16
	Offset uint64
}



func NewFile(block *types.Block) (*FileItem, error) {
	var err error
	//blockKey,_ := block.HashHex()
	blockKey := block.GetHead().GetHeight()

	fmt.Println("",blockKey)
	filesize :=  FILE_SIZE
	fileTypestr := ".topia"

	filepath := strconv.FormatInt(int64(blockKey), 10) + fileTypestr
	fmt.Println(filepath)
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil{
		panic(err)
	}
	_,err = file.Write(make([]byte, filesize))

	if err != nil {
		return nil, err
	}
	FileNameOpening = filepath

	tp := FileItem{
		Filetype: DataFileType,
		File:   file,
		Offset: FILE_HEADER_SIZE,
		Bloom: New(),
	}

	tp.Writedata(block)


	err = NewIndexFile(block)

	if err != nil{
		panic(err)
	}


	return &tp, nil
}



func NewIndexFile(block *types.Block) ( error) {
	blockKey := block.GetHead().GetHeight()
	filepath := strconv.FormatInt(int64(blockKey), 10) + ".index"

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, FILE_SIZE))
	if err != nil {
		return  err
	}

	//stat, err := os.Stat(filepath)
	//if err != nil {
	//	return nil, err
	//}

	var tp  = FileItem{
		Filetype: IndexFileType,
		File:   file,
		Offset: 0,

	}
	//what's the version ?????

	tp.Writeindex(88,0)


	return nil
}

//func NewTransFile(block *types.Block) ( error) {
//	blockKey := block.GetHead().GetHeight()
//	filepath := strconv.FormatInt(int64(blockKey), 10) + ".trans"
//
//	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//	file.Write(make([]byte, FILE_SIZE))
//	if err != nil {
//		return err
//	}
//
//	//stat, err := os.Stat(filepath)
//	//if err != nil {
//	//	return nil, err
//	//}
//
//	var tp  = FileItem{
//		Filetype: 2,
//		File:   file,
//		Offset: 0,
//	}
//
//	tp.Writetrans(block)
//
//	return nil
//}

