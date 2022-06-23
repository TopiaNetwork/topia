package block

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"
	//"encoding/json"

	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/snksoft/crc"
	"launchpad.net/gommap"
	//tplog "github.com/TopiaNetwork/topia/log"
)





func (df *FileItem) FindBlockbyNumber(blockNum types.BlockNum) (*types.Block, error) {

	var err error
	//indexfilename := filename ,".index")
	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock := df.File.Name()[:TraceIndex]
	fileindex, err := os.OpenFile(StartBlock+".index", os.O_RDWR, 0644)

	indexfile := FileItem{
		IndexFile,
		fileindex,
		FILE_HEADER_SIZE,
		0,
	}


	//first index
	tpindex, err := indexfile.Findindex(blockNum)

	fmt.Println(tpindex)

	dataoffset := tpindex.offset


	filedata, err := os.OpenFile(df.File.Name(), os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}

	datammap,err  := gommap.Map(filedata.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)

	if err != nil{
		panic(err)
	}
	version := binary.BigEndian.Uint32(datammap[dataoffset:dataoffset + 4])
	fmt.Println(version)
	offset := binary.BigEndian.Uint64(datammap[dataoffset+4:dataoffset + 12])
	fmt.Println(offset)
	size := binary.BigEndian.Uint64(datammap[dataoffset+12:dataoffset + 20])
	fmt.Println(size)
	crc := binary.BigEndian.Uint64(datammap[dataoffset+20:dataoffset + 28])
	fmt.Println(crc)
	block := Decodeblock(datammap[dataoffset+28:dataoffset + 28+ size])
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
	var err error
	versionbyte := Uint32ToBytes(block.GetHead().Version)
	fmt.Println(versionbyte)
	offsetbyte := Uint64ToBytes(df.Offset)

	mmap, err := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}

	defer mmap.UnsafeUnmap()


	buf,_ := Encodeblock(block)
	ccittCrc := crc.CalculateCRC(crc.CCITT, buf)
	crcbyte :=  Uint64ToBytes(ccittCrc)
	fmt.Println(ccittCrc)
	size := uint64(len(buf))
	sizebyte := Uint64ToBytes(size)

	copy(mmap[df.Offset:df.Offset+4],versionbyte)
	copy(mmap[df.Offset+4:df.Offset+12],offsetbyte)
	copy(mmap[df.Offset+12:df.Offset+20],sizebyte)
	copy(mmap[df.Offset+20:df.Offset+16],crcbyte)
	copy(mmap[df.Offset+16:df.Offset+16+size],buf)

	_ = df.File.Sync()
	df.Offset = df.Offset + 16 + size
	return  nil
}


func (df *FileItem) WriteHeader(block *types.Block) error {
	versionbyte := Uint32ToBytes(block.GetHead().Version)

	//
	txids := block.GetData().GetTxs()
	if txids == nil{
		return nil
	}

	blockKey := block.GetHead().GetHeight()

	index,err := df.Findindex(types.BlockNum(blockKey))

	if err != nil{
		panic(err)
	}



	mmap, err := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer mmap.UnsafeUnmap()
	txid_len := len(txids)
	for i:=0;i < txid_len;i++ {

		copy(mmap[df.Offset:df.Offset+2], versionbyte)
		copy(mmap[df.Offset+2:df.Offset+10],txids[i])
		copy(mmap[df.Offset+10:df.Offset+18], Uint64ToBytes(blockKey))
		copy(mmap[df.Offset+18:df.Offset+26], Uint64ToBytes(index.offset))
	}

	_ = df.File.Sync()
	Indexoffset = Indexoffset + 26
	df.Offset = df.Offset + 26

	return  nil

}



//func (df *FileItem) FindBlock(filename string) (*types.Block, error) {
//	//blockbyte,_ := json.Marshal(block)
//	//buf := make([]byte, len(blockbyte))
//
//	indexfilename := path.Join(filename ,".index")
//	file, err := os.OpenFile(indexfilename, os.O_RDWR, 0644)
//
//	//first index
//	indexmmap, _ := gommap.Map(file.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)
//
//	start := 0
//	end := len(indexmmap)
//
//	dataoffset,_ := binarySearch(start,end,indexmmap[0],indexmmap)
//
//	datafilename := path.Join(filename ,".topia")
//	filedata, err := os.OpenFile(datafilename, os.O_RDWR, 0644)
//	datammap, _ := gommap.Map(filedata.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)
//
//	block := Decodeblock(datammap[0:dataoffset])
//	//item, err := Decodeblock(buf)
//	if block.Size() < 0{
//		return nil,err
//	}
//
//
//	if err != nil {
//		return nil, err
//	}
//	return block,err
//
//}


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


func RollBackWrite(blockNum types.BlockNum) error{
	
	return nil
}




