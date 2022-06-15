package block

import (
	//"bytes"
	//"encoding/binary"
	//"golang.org/x/tools/go/ssa"

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

func (df *TopiaFile) FindBlock(filename string) (*types.Block, error) {
	//blockbyte,_ := json.Marshal(block)
	//buf := make([]byte, len(blockbyte))

	indexfilename := path.Join(filename ,".index")
	file, err := os.OpenFile(indexfilename, os.O_RDWR, 0644)

	//first index
	indexmmap, _ := gommap.Map(file.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	start := 0
	end := len(indexmmap)
	//二分查找
	dataoffset,_ := binarySearch(start,end,indexmmap[0],indexmmap)

	datafilename := path.Join(filename ,".topia")
	filedata, err := os.OpenFile(datafilename, os.O_RDWR, 0644)
	datammap, _ := gommap.Map(filedata.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	block,_ := Decodeblock(datammap[0:dataoffset])
	//item, err := Decodeblock(buf)
	if block.Size() < 0{
		return nil,err
	}


	if err != nil {
		return nil, err
	}
	return block,err

}


func (df *TopiaFile) findTrans(filename string) (*types.Block, error) {
	//blockbyte,_ := json.Marshal(block)
	//buf := make([]byte, len(blockbyte))

	indexfilename := path.Join(filename ,".trans")
	file, err := os.OpenFile(indexfilename, os.O_RDWR, 0644)

	//first index
	indexmmap, _ := gommap.Map(file.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	start := 0
	end := len(indexmmap)
	//二分查找
	dataoffset,_ := binarySearch(start,end,indexmmap[0],indexmmap)

	datafilename := path.Join(filename ,".topia")
	filedata, err := os.OpenFile(datafilename, os.O_RDWR, 0644)
	datammap, _ := gommap.Map(filedata.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	block,_ := Decodeblock(datammap[0:dataoffset])
	//item, err := Decodeblock(buf)
	if block.Size() < 0{
		return nil,err
	}


	if err != nil {
		return nil, err
	}
	return block,err

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

func binarySearch(start int, end int,blockid byte,mmap gommap.MMap)(int,bool){
	//current := end / 2
	//for end-start > 1 {
	//	compareWithCurrentWord := bytes.Compare(blockid,string(mmap[current]) )
	//	compareWithCurrentWord == 0 {
	//		return current, true
	//		} else if compareWithCurrentWord < 0 {
	//		end = current
	//		current = (start + current) / 2
	//		} else {
	//		start = current
	//		current = (current + end) / 2
	//		     }
	//	}
	return end, false
	}

//func getFilename()(string){
//
//}
