package block

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"syscall"


	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
)





func (df *FileItem) Findindex(blockNum types.BlockNum) (*IndexItem, error) {

	TraceIndex := strings.Index(df.File.Name(), ".")
	StartBlock,_ := strconv.Atoi(df.File.Name()[:TraceIndex])
	//fmt.Println(StartBlock)

	indexnum := int16(blockNum) - int16(StartBlock)
	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()
	versionint := binary.BigEndian.Uint16(mmap[indexnum*6:indexnum*6+2])
	positionint := binary.BigEndian.Uint64(mmap[indexnum*6+2:indexnum*6+4])
	offsetint := binary.BigEndian.Uint64(mmap[indexnum*6+4:indexnum*6+6])

	tpindex := IndexItem{
		versionint,
		positionint,
		offsetint,
	}


	return  &tpindex, nil

}


func (df *FileItem) Writeindex(version uint16,offset uint64) error {
	//versionbyte,_ := json.Marshal(version)
	versionbyte := Uint16ToBytes(version)
	offsetbyte := Uint64ToBytes(offset)
	offsetindex := Uint64ToBytes(df.Offset)

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












