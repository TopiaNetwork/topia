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
		1,
		fileindex,
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
	var err error
	versionbyte := Int32ToBytes(int32(block.GetHead().Version))
	fmt.Println(versionbyte)
	offsetbyte := Int16ToBytes(df.Offset)

	mmap, err := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}

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




