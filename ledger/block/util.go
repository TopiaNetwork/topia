package block

import (
	"encoding/binary"
	"strings"

	//"encoding/json"

	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
	//tplog "github.com/TopiaNetwork/topia/log"
)

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

func Uint16ToBytes(i uint16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, i)
	return buf
}

func Uint32ToBytes(i uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, i)
	return buf
}

func Uint64ToBytes(i uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	return buf
}

func GetStartblockFromFilename(file *FileItem) string{
	TraceIndex := strings.Index(file.File.Name(), ".")
	StartBlock := file.File.Name()[:TraceIndex]
	return StartBlock
}

func GetIndexFilename(file *FileItem) string{
	StartBlock := GetStartblockFromFilename(file)
	return StartBlock + ".index"
}


func GetDataFilename(file *FileItem) string{
	StartBlock := GetStartblockFromFilename(file)
	return StartBlock + ".data"
}

func Getmmap(file *FileItem)