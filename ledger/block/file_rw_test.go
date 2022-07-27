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


func TestWritedata(t *testing.T) {
	file, err := os.OpenFile("test.topia", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = FileItem{
		1,
		file,
		FILE_HEADER_SIZE,
		0,
		nil,

	}
	err =to.Writedata(&block_all)
	if err != nil{
		panic(err)
	}
}

//func TestFileItem_Writeindex(t *testing.T) {
//	file, err := os.OpenFile("test.topia", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//
//	if err != nil{
//		panic(err)
//	}
//
//	var to = FileItem{
//		1,
//		file,
//		1,
//		1,
//		nil,
//
//	}
//	err =to.Writedata(&block_all)
//	if err != nil{
//		panic(err)
//	}
//}

