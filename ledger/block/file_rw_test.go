package block

import (
	"os"
	"testing"
)

func TestReaddata(t *testing.T) {
	filename := FilenameNow + ".topia"
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = FileItem{
		1,
		file,
		1,
		1,
		nil,

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
		1,
		1,
		nil,

	}
	err =to.Writedata(&block_all)
	if err != nil{
		panic(err)
	}
}

func TestFileItem_Writeindex(t *testing.T) {
	file, err := os.OpenFile("test.topia", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = FileItem{
		1,
		file,
		1,
		1,
		nil,

	}
	err =to.Writedata(&block_all)
	if err != nil{
		panic(err)
	}
}

