package block

import (
	//"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"launchpad.net/gommap"
	"syscall"

	"os"
	"path"
	//"syscall"
)

const LOCK_FILE = "LOCK"
const DATA_FILE = "DATA"
const MERGE_FILE = "DATA.MER"
const FILE_SIZE = 10000000


var FileNameOpening = ""

type TopiaFile struct {
	File   *os.File
	Offset int64
}

//func newTopiaFile(basePath string) (*TopiaFile, error) {
//	datafile := basePath + string(os.PathSeparator) + DATA_FILE
//	return newFileImpl(datafile)
//}


func newDataFile(block *types.Block) (*TopiaFile, error) {

	blockKey,_ := block.HashHex()
	filepath := path.Join(blockKey ,".topia")

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
		Offset: stat.Size(),
	}, nil
}

func newIndexFile(block *types.Block) (*TopiaFile, error) {
	blockKey,_ := block.HashHex()
	filepath := path.Join(blockKey ,".index")

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	return &TopiaFile{
		File:   file,
		Offset: stat.Size(),
	}, nil
}

func (df *TopiaFile) Writedata(block *types.Block) error {

	mmap, _ := gommap.MapAt(0,file.Fd(), 0,100,syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	defer mmap.UnsafeUnmap()


	newFsConfigBytes, _ := json.Marshal(block)
	//_, err = file.Seek(int64(len(newFsConfigBytes)), 1)
	//if err != nil {
	//	log.Fatal("Failed to seek")
	//}
	for i:=0;i<len(newFsConfigBytes);i++ {
		mmap[i] = newFsConfigBytes[i]
		mmap.Sync(syscall.MS_SYNC)

		err = file.Sync()

		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println("", i)
	}
	return  nil
}

//func (df *TopiaFile) ReadItem(offset int64) (*DbItem, error) {
//	buf := make([]byte, DbItemHdrSize)
//	if _, err := gommap.Map(f.Fd(), gommap.PROT_READ, gommap.MAP_PRIVATE)
//
//	if err != nil {
//		return nil, err
//	}
//
//
//}

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

func getFilename()(string){

}
