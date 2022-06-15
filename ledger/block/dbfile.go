package block

import (
	"encoding/binary"
	//"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
	"syscall"
	"encoding/json"
	"github.com/snksoft/crc"

	"os"
	"path"
	//"syscall"
)

const LOCK_FILE = "LOCK"
const DATA_FILE = "DATA"

const FILE_SIZE = 10000000



var FileNameOpening = ""
var Indexoffset = 0
var Transoffset = 0

type TopiaFile struct {
	Filetype int8 //0,data;1,index;2,transactionindex
	File   *os.File
	Offset int
}

//func newTopiaFile(basePath string) (*TopiaFile, error) {
//	datafile := basePath + string(os.PathSeparator) + DATA_FILE
//	return newFileImpl(datafile)
//}


func NewFile(block *types.Block,filetype int8) (*TopiaFile, error) {

	blockKey,_ := block.HashHex()
	filetypestr := ".topai"
	switch filetype {
	case 1:
		filetypestr = ".index"
	case 2:
		filetypestr = ".trans"
	}
	filepath := path.Join(blockKey ,filetypestr)

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	FileNameOpening = filepath

	stat, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	return &TopiaFile{
		File:   file,
		Offset: int(stat.Size()),
		Filetype: filetype,
	}, nil
}

//func newIndexFile(block *types.Block) (*TopiaFile, error) {
//	blockKey,_ := block.HashHex()
//	filepath := path.Join(blockKey ,".index")
//
//	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//	if err != nil {
//		return nil, err
//	}
//
//	stat, err := os.Stat(filepath)
//	if err != nil {
//		return nil, err
//	}
//
//	return &TopiaFile{
//		File:   file,
//		Offset: int(stat.Size()),
//	}, nil
//}


func (df *TopiaFile) Writedata(block *types.Block) error {
	version,_ := json.Marshal(block.GetData().Version)
	offerbyte,_ := json.Marshal(df.Offset)

	df.File.Write(version)
	df.File.Write(offerbyte)



	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()


	newFsConfigBytes, _ := json.Marshal(block)
	ccittCrc := crc.CalculateCRC(crc.CCITT, newFsConfigBytes)
	crcbyte,_ :=  json.Marshal(ccittCrc)
	size := len(newFsConfigBytes) + len(crcbyte) + 3
	sizebyte,_ := json.Marshal(size)
	df.File.Write(sizebyte)

	j := 0
	for i:=0;i<len(newFsConfigBytes);i++{
		mmap[j] = newFsConfigBytes[i]
		mmap.Sync(syscall.MS_SYNC)

		j++
	}

	for i:=0;i<len(crcbyte);i++ {
		mmap[j+1] = crcbyte[i]
		mmap.Sync(syscall.MS_SYNC)
		j++
	}

	_ = df.File.Sync()
	df.Offset += size
	return  nil
}

func (df *TopiaFile) Writeindex(version int,offset int) error {
	versionbyte,_ := json.Marshal(version)
	offsetbyte,_ := json.Marshal(offset)
	offsetindex,_ := json.Marshal(Indexoffset+3)


	df.File.Write(versionbyte)
	df.File.Write(offsetbyte)
	df.File.Write(offsetindex)



	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()


	_ = df.File.Sync()
	Indexoffset = Indexoffset + 3
	return  nil

}

func (df *TopiaFile) Writetrans(version int,txid string,blockheight int, offset int) error {
	versionbyte,_ := json.Marshal(version)
	txidbyte,_ := json.Marshal(txid)
	blockheightbyte,_ := json.Marshal(blockheight)
	offsetbyte,_ := json.Marshal(offset)



	df.File.Write(versionbyte)
	df.File.Write(txidbyte)
	df.File.Write(blockheightbyte)
	df.File.Write(offsetbyte)



	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()


	_ = df.File.Sync()
	Indexoffset = Indexoffset + 3
	return  nil

}

func (df *TopiaFile) ReadItem(block *types.Block) (*types.Block, error) {
	blockbyte,_ := json.Marshal(block)
	buf := make([]byte, len(blockbyte))

	bufmap := gommap.Map(df.File.Fd(), gommap.PROT_READ, gommap.MAP_PRIVATE)

	item, err := Decodeblock(buf)
	if item.Size() < 0{
		return nil,err
	}


	if err != nil {
		return nil, err
	}


}

func outSize(filepath string)(bool){
	if FileNameOpening == ""{
		return true
	}

	stat, err := os.Stat(filepath)
	if err != nil {
		return false
	}
	if stat.Size() >  FILE_SIZE{
		return true
	}
	return false
}

func Decodeblock(buf []byte) (*types.Block, error) {
	//ks := binary.BigEndian.Uint64(buf[0:8])
	//vs := binary.BigEndian.Uint64(buf[8:16])
	//flag := binary.BigEndian.Uint32(buf[16:20])
	//return &types.Block{Head: ks, Data: vs}, nil
	return nil,nil
}

//func getFilename()(string){
//
//}
