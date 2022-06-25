package block

import (
	"encoding/binary"
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
	"os"
	"strings"
	"syscall"
)


type BackItem struct{
	version uint16
	filetype FileType
	startoffset uint64
	size uint64
}


func (df *FileItem) RollBackIndex() (error) {

	var err error

	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock := df.File.Name()[:TraceIndex]
	fileback, err := os.OpenFile(StartBlock+".back", os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}

	datammap, err := gommap.Map(fileback.Fd(), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer datammap.UnsafeUnmap()

	fileindex, err := os.OpenFile(StartBlock+".index", os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}

	indexmmap, err := gommap.Map(fileindex.Fd(), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer indexmmap.UnsafeUnmap()

	var byte0 []byte
	for i :=0; i*20 < int(df.HeaderOffset); i++ {
		filetype := binary.BigEndian.Uint16(datammap[i*20+2:i*20 + 4])
		if  FileType(filetype) == IndexFileType{
			startindex := binary.BigEndian.Uint64(datammap[i*20+4:i*20 + 12])
			endindex := binary.BigEndian.Uint64(datammap[i*20+12:i*20 + 20])

			copy(indexmmap[startindex:endindex],byte0)
		}
	}

	return nil



}

func (df *FileItem) RollBackHeader() ( error) {
	var err error

	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock := df.File.Name()[:TraceIndex]
	fileback, err := os.OpenFile(StartBlock+".back", os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}

	datammap, err := gommap.Map(fileback.Fd(), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer datammap.UnsafeUnmap()

	fileindex, err := os.OpenFile(StartBlock+".topia", os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}

	indexmmap, err := gommap.Map(fileindex.Fd(), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer indexmmap.UnsafeUnmap()

	var byte0 []byte
	for i :=0; i*20 < int(df.HeaderOffset); i++ {
		filetype := binary.BigEndian.Uint16(datammap[i*20+2:i*20 + 4])
		if  FileType(filetype) == DataHeaderType{
			startindex := binary.BigEndian.Uint64(datammap[i*20+4:i*20 + 12])
			endindex := binary.BigEndian.Uint64(datammap[i*20+12:i*20 + 20])

			copy(indexmmap[startindex:endindex],byte0)
		}
	}

	return nil
}

func (df *FileItem) RollBackData() (error) {
	var err error

	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock := df.File.Name()[:TraceIndex]
	fileback, err := os.OpenFile(StartBlock+".back", os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}

	datammap, err := gommap.Map(fileback.Fd(), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer datammap.UnsafeUnmap()

	fileindex, err := os.OpenFile(StartBlock+".topia", os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}

	indexmmap, err := gommap.Map(fileindex.Fd(), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer indexmmap.UnsafeUnmap()

	var byte0 []byte
	for i :=0; i*20 < int(df.HeaderOffset); i++ {
		filetype := binary.BigEndian.Uint16(datammap[i*20+2:i*20 + 4])
		if  FileType(filetype) == DataFileType{
			startindex := binary.BigEndian.Uint64(datammap[i*20+4:i*20 + 12])
			endindex := binary.BigEndian.Uint64(datammap[i*20+12:i*20 + 20])

			copy(indexmmap[startindex:endindex],byte0)
		}
	}

	return nil
}

func (df *FileItem) EmptyRoll() (error) {

	var err error

	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock := df.File.Name()[:TraceIndex]
	fileback, err := os.OpenFile(StartBlock+".back", os.O_RDWR, 0644)
	if err != nil{
		panic(err)
	}

	datammap, err := gommap.Map(fileback.Fd(), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}
	defer datammap.UnsafeUnmap()

	var byte0 []byte
	copy(datammap[0:df.Offset],byte0)

	return nil
}

func (df *FileItem)RollBackWrite(startoffset uint64,size uint64, block *types.Block) error{

	var err error
	//version type the same as block
	versionbyte := Uint32ToBytes(block.GetHead().Version)
	fmt.Println(versionbyte)
	filetypebyte := Uint16ToBytes(uint16(df.Filetype))

	mmap, err := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil{
		panic(err)
	}

	defer mmap.UnsafeUnmap()

	startbyte := Uint64ToBytes(startoffset)

	sizebyte := Uint64ToBytes(size)

	copy(mmap[df.Offset:df.Offset+2],versionbyte)
	copy(mmap[df.Offset+2:df.Offset+4],filetypebyte)
	copy(mmap[df.Offset+4:df.Offset+12],startbyte)
	copy(mmap[df.Offset+12:df.Offset+20],sizebyte)

	_ = df.File.Sync()
	df.Offset = df.Offset + 20
	return  nil
}