package block

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	//"encoding/json"

	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
	//tplog "github.com/TopiaNetwork/topia/log"
)





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












