package block

import (
	"fmt"
	"os"
	"strconv"
	//"encoding/json"

	"github.com/TopiaNetwork/topia/chain/types"
	//tplog "github.com/TopiaNetwork/topia/log"
)


const FILE_SIZE = 10000 //* 1000

type FileType int8

const (
	DataFile       FileType = 0
	IndexFile      FileType = 1
)

var FileNameOpening = ""
var Indexoffset = 0
//var Transoffset = 0

type FileItem struct {
	//header
	Filetype FileType //0,data;1,index;2,transactionindex
	File   *os.File
	Offset int16 //INT64
}


type DataItem struct{
	version int32
	offset int64
	size int16
	crc int64
	data *types.Block
}


type IndexItem struct{
	version int16
	position int16
	offset int64
}


type TransIndex struct{
	Version int16
	Txid int64
	BlockHeight int16
	Offset int64
}
//func newFileItem(basePath string) (*FileItem, error) {
//	datafile := basePath + string(os.PathSeparator) + DATA_FILE
//	return newFileImpl(datafile)
//}


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
		Filetype: DataFile,
		File:   file,
		Offset: 0,
	}

	tp.Writedata(block)


	err = NewIndexFile(block)

	if err != nil{
		panic(err)
	}


	return &tp, nil
}

// transaction to bytes
//func (m *Transaction) HashBytes() ([]byte, error) {
//	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
//	txBytes, err := marshaler.Marshal(m)
//	if err != nil {
//		return nil, err
//	}
//
//	hasher := tpcmm.NewBlake2bHasher(0)
//
//	return hasher.Compute(string(txBytes)), nil
//}

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
		Filetype: IndexFile,
		File:   file,
		Offset: 0,
	}
	//what's the version ?????
	//索引是哪个版本的再哪确定
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

