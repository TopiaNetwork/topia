package block

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	//"encoding/json"

	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/snksoft/crc"
	"launchpad.net/gommap"
)


const FILE_SIZE = 10000 //* 1000



var FileNameOpening = ""
var Indexoffset = 0
var Transoffset = 0

type FileItem struct {
	//header
	Filetype int8 //0,data;1,index;2,transactionindex
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

	//blockKey,_ := block.HashHex()
	blockKey := block.GetHead().GetHeight()

	fmt.Println("",blockKey)
	filesize :=  FILE_SIZE
	fileTypestr := ".topia"

	filepath := strconv.FormatInt(int64(blockKey), 10) + fileTypestr
	fmt.Println(filepath)
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, filesize))

	if err != nil {
		return nil, err
	}
	FileNameOpening = filepath

	tp := FileItem{
		Filetype: 0,
		File:   file,
		Offset: 0,
	}

	tp.Writedata(block)


	NewIndexFile(block)

	NewTransFile(block)

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

func NewIndexFile(block *types.Block) (*FileItem, error) {
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

	var tp  = FileItem{
		Filetype: 1,
		File:   file,
		Offset: 0,
	}
	//what's the version ?????
	//索引是哪个版本的再哪确定
	tp.Writeindex(88,0)


	return &tp, nil
}

func NewTransFile(block *types.Block) (*FileItem, error) {
	blockKey := block.GetHead().GetHeight()
	filepath := strconv.FormatInt(int64(blockKey), 10) + ".trans"

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	file.Write(make([]byte, FILE_SIZE))
	if err != nil {
		return nil, err
	}

	//stat, err := os.Stat(filepath)
	//if err != nil {
	//	return nil, err
	//}

	var tp  = FileItem{
		Filetype: 2,
		File:   file,
		Offset: 0,
	}

	tp.Writetrans(block)

	return &tp, nil
}


func (df *FileItem) FindBlockbyNumber(blockNum types.BlockNum) (*types.Block, error) {


	//indexfilename := filename ,".index")
	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock := df.File.Name()[:TraceIndex]
	fileindex, err := os.OpenFile(StartBlock+".index", os.O_RDWR, 0644)

	indexfile := FileItem{
		1,
		fileindex,
		0,
	}


	//first index
	tpindex, _ := indexfile.Findindex(blockNum)

	fmt.Println(tpindex)

	dataoffset := tpindex.offset


	filedata, err := os.OpenFile(df.File.Name(), os.O_RDWR, 0644)
	datammap, _ := gommap.Map(filedata.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	version := int32(binary.BigEndian.Uint32(datammap[dataoffset:dataoffset + 4]))
	fmt.Println(version)
	offset := int16(binary.BigEndian.Uint16(datammap[dataoffset+4:dataoffset + 6]))
	fmt.Println(offset)
	size := int16(binary.BigEndian.Uint16(datammap[dataoffset+6:dataoffset + 8]))
	fmt.Println(size)
	crc := int64(binary.BigEndian.Uint64(datammap[dataoffset+8:dataoffset + 16]))
	fmt.Println(crc)
	block := Decodeblock(datammap[dataoffset+16:dataoffset + 16+ int64(size)])
	fmt.Println(block)
	//item, err := Decodeblock(buf)
	if block.Size() < 0{
		return nil,err
	}


	if err != nil {
		return nil, err
	}
	return block,err

}

func (df *FileItem) Writedata(block *types.Block) error {
	versionbyte := Int32ToBytes(int32(block.GetHead().Version))
	fmt.Println(versionbyte)
	offsetbyte := Int16ToBytes(df.Offset)

	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)

	defer mmap.UnsafeUnmap()


	buf,_ := Encodeblock(block)
	ccittCrc := crc.CalculateCRC(crc.CCITT, buf)
	crcbyte :=  Int64ToBytes(int64(ccittCrc))
	fmt.Println(ccittCrc)
	size := int16(len(buf))
	sizebyte := Int16ToBytes(size)

	copy(mmap[df.Offset:df.Offset+4],versionbyte)
	copy(mmap[df.Offset+4:df.Offset+6],offsetbyte)
	copy(mmap[df.Offset+6:df.Offset+8],sizebyte)
	copy(mmap[df.Offset+8:df.Offset+16],crcbyte)
	copy(mmap[df.Offset+16:df.Offset+16+size],buf)

	_ = df.File.Sync()
	df.Offset = df.Offset + 16 + size
	return  nil
}



func (df *FileItem) Findindex(blockNum types.BlockNum) (*IndexItem, error) {

	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock,_ := strconv.Atoi(df.File.Name()[:TraceIndex])
	//fmt.Println(StartBlock)

	indexnum := int16(blockNum) - int16(StartBlock)
	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()
	versionint := int16(binary.BigEndian.Uint16(mmap[indexnum*6:indexnum*6+2]))
	positionint := int16(binary.BigEndian.Uint16(mmap[indexnum*6+2:indexnum*6+4]))
	offsetint := int64(binary.BigEndian.Uint16(mmap[indexnum*6+4:indexnum*6+6]))

	tpindex := IndexItem{
		versionint,
		positionint,
		offsetint,
	}


	return  &tpindex, nil

}


func (df *FileItem) Writeindex(version int16,offset int16) error {
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

func (df *FileItem) Writetrans(block *types.Block) error {
	versionbyte := Int32ToBytes(int32(block.GetHead().Version))

	//
	txids := block.GetData().GetTxs()

	blockKey := block.GetHead().GetHeight()


	versionint := int16(binary.BigEndian.Uint16(versionbyte))
	fmt.Println(versionint)

	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()
	for txid := range txids {

		copy(mmap[df.Offset:df.Offset+2], versionbyte)
		copy(mmap[df.Offset+2:df.Offset+10],Int64ToBytes(int64(txid)))
		copy(mmap[df.Offset+4:df.Offset+6], Int64ToBytes(int64(blockKey)))
		//

	}

	_ = df.File.Sync()
	Indexoffset = Indexoffset + 6
	df.Offset = df.Offset + 6

	return  nil

}



func (df *FileItem) FindBlock(filename string) (*types.Block, error) {
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


func (df *FileItem) findTrans(filename string) (*types.Block, error) {
	//blockbyte,_ := json.Marshal(block)
	//buf := make([]byte, len(blockbyte))

	indexfilename := path.Join(filename ,".trans")
	file, err := os.OpenFile(indexfilename, os.O_RDWR, 0644)

	//first index
	indexmmap, _ := gommap.Map(file.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	start := 0
	end := len(indexmmap)

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
