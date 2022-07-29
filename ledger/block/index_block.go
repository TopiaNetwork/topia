package block

import (
	"encoding/binary"
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

	indexnum := uint64(blockNum) - uint64(StartBlock)
	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()
	versionint := binary.BigEndian.Uint16(mmap[indexnum*18:indexnum*18+2])
	positionint := binary.BigEndian.Uint64(mmap[indexnum*18+2:indexnum*18+10])
	offsetint := binary.BigEndian.Uint64(mmap[indexnum*18+10:indexnum*18+18])

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

	//fmt.Println(versionbyte)
	//fmt.Println("",offsetbyte)
	//fmt.Println("",offsetindex)

	//versionint := int16(binary.BigEndian.Uint16(versionbyte))
	//fmt.Println(versionint)
	//
	mmap, _ := gommap.Map(df.File.Fd(),syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()
	copy(mmap[df.Offset:df.Offset+2],versionbyte)
	copy(mmap[df.Offset+2:df.Offset+10],offsetbyte)
	copy(mmap[df.Offset+10:df.Offset+18],offsetindex)
	//diffrent offset and position

	_ = df.File.Sync()
	//Indexoffset = Indexoffset + 18
	df.Offset = df.Offset + 18
	return  nil
}












