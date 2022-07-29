package block

import (
	"fmt"
	"os"
	"testing"
)

func TestReaddata(t *testing.T) {
	filename := FilenameNow + ".topia"
	fmt.Println(filename)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = FileItem{
		1,
		file,
		FILE_HEADER_SIZE,
		0,
		New(),
	}
	err =to.Writedata(&block_all)
	if err != nil{
		panic(err)
	}
}


//func TestWritedata(t *testing.T) {
//	file, err := os.OpenFile("123456.topia", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//
//	if err != nil{
//		panic(err)
//	}
//
//	var to = FileItem{
//		1,
//		file,
//		FILE_HEADER_SIZE,
//		0,
//		nil,
//
//	}
//	err =to.Writedata(&block_all)
//	if err != nil{
//		panic(err)
//	}
//}
//
//func TestFileItem_Writeindex(t *testing.T) {
//	file, err := os.OpenFile("123456.index", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//
//	if err != nil{
//		panic(err)
//	}
//
//	var to = FileItem{
//		1,
//		file,
//		0,
//		0,
//		nil,
//
//	}
//	err =to.Writeindex(1,1)
//	if err != nil{
//		panic(err)
//	}
//}

