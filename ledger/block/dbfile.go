package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"os"
	"strconv"
	"sync"
)


const FILE_SIZE = 10000 * 1000
const FILE_HEADER_SIZE = 10000
const HEADER_SLICE_SIZE = 26
const INDEX_SLICE_SIZE = 18

type FileType uint16

const (
	DataFileType       FileType = 0
	IndexFileType      FileType = 1
	//DataHeaderType      FileType = 2
	RollbackFileType      FileType = 2
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

var DatafileSingle *FileItem
var IndexfileSingle *FileItem
var RollfileSigle *FileItem
var lock = &sync.Mutex{}

func NewFile(block *types.Block) (*FileItem, error) {
	var err error
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
		DataFileType,
		file,
		FILE_HEADER_SIZE,
		0,
		New(),
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
	filepath := strconv.FormatUint(blockKey, 10) + ".index"

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, FILE_SIZE))
	if err != nil {
		return  err
	}

	var tp  = FileItem{
		IndexFileType,
		file,
		FILE_HEADER_SIZE,
		0,
		nil,

	}

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

func getDatafile(filename string) *FileItem {

	if DatafileSingle == nil {
		lock.Lock()
		defer lock.Unlock()
		if DatafileSingle == nil {
			fmt.Println("Creating single instance now.")
			DatafileSingle = newFileitem(filename,0)
		} else {
			fmt.Println("Single instance already created.")
		}
	} else {
		fmt.Println("Single instance already created.")
	}

	return DatafileSingle
}

func getIndexfile(filename string) *FileItem {

	if IndexfileSingle == nil {
		lock.Lock()
		defer lock.Unlock()
		if IndexfileSingle == nil {
			fmt.Println("Creating single instance now.")
			IndexfileSingle = newFileitem(filename,1)
		} else {
			fmt.Println("Single instance already created.")
		}
	} else {
		fmt.Println("Single instance already created.")
	}

	return IndexfileSingle
}

func getRollfile(filename string) *FileItem {

	if IndexfileSingle == nil {
		lock.Lock()
		defer lock.Unlock()
		if IndexfileSingle == nil {
			fmt.Println("Creating single instance now.")
			IndexfileSingle = newFileitem(filename,2)
		} else {
			fmt.Println("Single instance already created.")
		}
	} else {
		fmt.Println("Single instance already created.")
	}

	return IndexfileSingle
}


func newFileitem(filename string,filetype FileType)*FileItem{
	suffix,offset := GetFilesuffix(filetype)

	file, _ := os.OpenFile(filename+suffix, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, FILE_SIZE))
	//if err != nil {
	//	return  nil
	//}

	var tp  = FileItem{
		filetype,
		file,
		offset,
		0,
		New(),
	}
	return &tp
}


func newtestfile(filename string,filetype FileType)FileItem{
	suffix,offset := GetFilesuffix(filetype)

	file, _ := os.OpenFile(filename+suffix, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, FILE_SIZE))
	//if err != nil {
	//	return  nil
	//}

	var tp  = FileItem{
		filetype,
		file,
		offset,
		0,
		New(),
	}
	return tp
}