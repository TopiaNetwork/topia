package block

import (

	"os"
	"path"
	"fmt"
	"syscall"
	"strconv"
	"strings"
	"encoding/binary"
	"encoding/json"


	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/snksoft/crc"
	"launchpad.net/gommap"

)

const LOCK_FILE = "LOCK"
const DATA_FILE = "DATA"

const FILE_SIZE = 10000 //* 1000



var FileNameOpening = ""
var Indexoffset = 0
var Transoffset = 0

type TopiaFile struct {
	Filetype int8 //0,data;1,index;2,transactionindex
	File   *os.File
	Offset int16
}
type TopiaData struct{
	version int32
	offset int16
	size int16
	crc int64
	data *types.Block
}
type TopiaIndex struct{
	version int16
	position int16
	offset int16
}

type TransIndex struct{
	Version int16
	Txid int64
	BlockHeight int16
	Offset int16
}
//func newTopiaFile(basePath string) (*TopiaFile, error) {
//	datafile := basePath + string(os.PathSeparator) + DATA_FILE
//	return newFileImpl(datafile)
//}


func NewFile(block *types.Block,filetype int8) (*TopiaFile, error) {

	//blockKey,_ := block.HashHex()
	blockKey := block.GetHead().GetHeight()

	fmt.Println("",blockKey)
	filesize :=  FILE_SIZE
	fileTypestr := ".topai"
	//switch filetype {
	//case 1:
	//	filetypestr = ".index"
	//	filesize = FILE_SIZE/2
	//case 2:
	//	filetypestr = ".trans"
	//	filesize = FILE_SIZE/2
	//}
	filepath := strconv.FormatInt(int64(blockKey), 10) + fileTypestr
	fmt.Println("",filepath)
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, filesize))

	if err != nil {
		return nil, err
	}
	FileNameOpening = filepath

	mmap, _ := gommap.Map(file.Fd(), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()

	buf,_ := Encodeblock(block)
	copy(mmap[:len(buf)],buf)


	//stat, err := os.Stat(filepath)
	//if err != nil {
	//	return nil, err
	//}

	NewIndexFile(block)

	return &TopiaFile{
		Filetype: filetype,
		File:   file,
		Offset: 0,
	}, nil
}


func NewIndexFile(block *types.Block) (*TopiaFile, error) {
	blockKey := block.GetHead().GetHeight()
	filepath := strconv.FormatInt(int64(blockKey), 10) + ".index"

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, FILE_SIZE))
	if err != nil {
		return nil, err
	}

	//stat, err := os.Stat(filepath)
	//if err != nil {
	//	return nil, err
	//}

	var tp  = TopiaFile{
		Filetype: 1,
		File:   file,
		Offset: 0,
	}
	//what's the version ?????
	tp.Writeindex(88,77)

	return &tp, nil
}

//func (df *TopiaFile) Readdata()
func (df *TopiaFile) FindBlockbyNumber(blockNum types.BlockNum) (*types.Block, error) {


	//indexfilename := filename ,".index")
	file, err := os.OpenFile(df.File.Name(), os.O_RDWR, 0644)

	//first index
	indexmmap, _ := gommap.Map(file.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	start := 0
	end := len(indexmmap)
	//二分查找
	dataoffset,_ := binarySearch(start,end,indexmmap[0],indexmmap)

	datafilename := df.File.Name() + ".topia"
	filedata, err := os.OpenFile(datafilename, os.O_RDWR, 0644)
	datammap, _ := gommap.Map(filedata.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	block := Decodeblock(datammap[0:dataoffset])
	//item, err := Decodeblock(buf)
	if block.Size() < 0{
		return nil,err
	}


	if err != nil {
		return nil, err
	}
	return block,err

}

func (df *TopiaFile) Writedata(block *types.Block) error {
	versionbyte := Int32ToBytes(int32(block.GetData().Version))
	offsetbyte := Int16ToBytes(df.Offset)

	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()


	buf,_ := Encodeblock(block)
	ccittCrc := crc.CalculateCRC(crc.CCITT, buf)
	crcbyte :=  Int64ToBytes(int64(ccittCrc))
	size := int16(len(buf))
	sizebyte := Int16ToBytes(size)

	copy(mmap[df.Offset:df.Offset+4],versionbyte)
	copy(mmap[df.Offset+4:df.Offset+6],offsetbyte)
	copy(mmap[df.Offset+6:df.Offset+8],sizebyte)
	copy(mmap[df.Offset+8:df.Offset+16],crcbyte)
	copy(mmap[df.Offset+16:df.Offset+16+size],crcbyte)



	_ = df.File.Sync()
	df.Offset = df.Offset + 16 + size
	return  nil
}

//func (df *TopiaFile) Writedata(block *types.Block) error {
//	version,_ := json.Marshal(block.GetData().Version)
//	offerbyte,_ := json.Marshal(df.Offset)
//
//	df.File.Write(version)
//	df.File.Write(offerbyte)
//
//
//
//	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
//	defer mmap.UnsafeUnmap()
//
//
//	newFsConfigBytes, _ := json.Marshal(block)
//	ccittCrc := crc.CalculateCRC(crc.CCITT, newFsConfigBytes)
//	crcbyte,_ :=  json.Marshal(ccittCrc)
//	size := len(newFsConfigBytes) + len(crcbyte) + 3
//	sizebyte,_ := json.Marshal(size)
//	df.File.Write(sizebyte)
//
//	j := 0
//	for i:=0;i<len(newFsConfigBytes);i++{
//		mmap[j] = newFsConfigBytes[i]
//		mmap.Sync(syscall.MS_SYNC)
//
//		j++
//	}
//
//	for i:=0;i<len(crcbyte);i++ {
//		mmap[j+1] = crcbyte[i]
//		mmap.Sync(syscall.MS_SYNC)
//		j++
//	}
//
//	_ = df.File.Sync()
//	df.Offset += size
//	return  nil
//}

func (df *TopiaFile) Findindex(blockNum types.BlockNum) (*TopiaIndex, error) {

	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock,_ := strconv.Atoi(df.File.Name()[:TraceIndex])
	//fmt.Println(StartBlock)

	indexnum := int16(blockNum) - int16(StartBlock)
	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()
	versionint := int16(binary.BigEndian.Uint16(mmap[indexnum*6:indexnum*6+2]))
	positionint := int16(binary.BigEndian.Uint16(mmap[indexnum*6+2:indexnum*6+4]))
	offsetint := int16(binary.BigEndian.Uint16(mmap[indexnum*6+4:indexnum*6+6]))

	tpindex := TopiaIndex{
		versionint,
		positionint,
		offsetint,
	}


	return  &tpindex, nil

}


func (df *TopiaFile) Writeindex(version int16,offset int16) error {
	//versionbyte,_ := json.Marshal(version)
	versionbyte := Int16ToBytes(version)
	offsetbyte := Int16ToBytes(offset)
	offsetindex := Int16ToBytes(df.Offset)

	fmt.Println(versionbyte)
	fmt.Println("",offsetbyte)
	fmt.Println("",offsetindex)


	versionint := int16(binary.BigEndian.Uint16(versionbyte))
	fmt.Println(versionint)

	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()
	copy(mmap[df.Offset:df.Offset+2],versionbyte)
	copy(mmap[df.Offset+2:df.Offset+4],offsetbyte)
	copy(mmap[df.Offset+4:df.Offset+6],offsetindex)

	_ = df.File.Sync()
	Indexoffset = Indexoffset + 6
	df.Offset = df.Offset + 6
	return  nil

}

func (df *TopiaFile) Writetrans(version int16,txid string,blockheight int16, offset int16) error {
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

	block := Decodeblock(datammap[0:dataoffset])
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

	block := Decodeblock(datammap[0:dataoffset])
	//item, err := Decodeblock(buf)
	if block.Size() < 0{
		return nil,err
	}


	if err != nil {
		return nil, err
	}
	return block,err

}

func OutSize(filepath string)(bool){
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



func Encodeblock(block *types.Block)([]byte, error)  {
	buf,err:= block.Marshal()
	if err != nil{
		return nil,err
	}
	return buf,err
}

func Decodeblock(buf []byte)(*types.Block)  {
	var b types.Block
	err := b.Unmarshal(buf)
	if err != nil{
		return nil
	}
	return &b
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

func Int16ToBytes(i int16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(i))
	return buf
}

func Int32ToBytes(i int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func Int64ToBytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}
//func getFilename()(string){
//
//}
