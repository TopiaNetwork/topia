package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"testing"
	//"github.com/fatih/structs"
	//"reflecting"

	//"fmt"
	//"github.com/TopiaNetwork/topia/chain/types"
)

var blockhead1 = types.BlockHead{
	nil,
	5,
	123456,
	0,
	1,
	nil,
	nil,
	nil,
	nil,
	nil,
	1,
	nil,
	nil,
	18,
	nil,
	nil,
	nil,
	nil,
	0,
	0,
	nil,
	nil,
	struct{}{} ,
	nil,
	20,
}
var txddata = [][]byte{
	{0,0,0,0,0},
	{1,1,1,1,1},
	{0,0,0,0,0},
	{0,0,0,0,0},
	{0,0,0,0,0},
}
var blockdata1 = types.BlockData{
	6,
	txddata,
	struct{}{},
	nil,
	1,
}

var block_all = types.Block{
	&blockhead1,
	&blockdata1,
	struct{}{},
	nil,
	100,
}

var FilenameNow = "1817128"
//struct block1{
//
//}



//func TestOutSize(t *testing.T) {
//
//	b := OutSize("text.txt")
//	fmt.Printf("%t",b)
//
//
//}

func TestEncodeblock(t *testing.T) {
	res,err := Encodeblock(&block_all)
	if err != nil{
		panic(err)

	}
	fmt.Println("encode block ")
	fmt.Println("",res)

	b := Decodeblock(res)
	fmt.Println("",b)
}

//func TestNewFile(t *testing.T) {
//	topia,_ := NewFile(&block_all)
//	fmt.Println("",topia)
//
//}

//

//
//func TestTopiaFile_Findindex(t *testing.T) {
//	filename := FilenameNow + ".index"
//	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
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
//	}
//	fmt.Println("tpindex: ",to)
//}


//func TestWriteindex(t *testing.T) {
//	filename := FilenameNow + ".topia"
//	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
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

//func TestFileItem_WriteHeader(t *testing.T) {
//	filename := FilenameNow + ".topia"
//	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
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
//	}
//	err =to.WriteHeader(&block_all)
//	if err != nil{
//		panic(err)
//	}
//}

//
//
//func TestWritetrans(t *testing.T) {
//	file, err := os.OpenFile("test.trans", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
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
//
//func TestFindBlock(t *testing.T) {
//
//	filename := FilenameNow + ".topia"
//	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//
//	if err != nil{
//		panic(err)
//	}
//
//	var to = FileItem{
//		0,
//		file,
//		0,
//		0,
//		nil,
//
//	}
//	tpdata,_ :=to.FindBlockbyNumber(1817128)
//	fmt.Println(tpdata)
//}

func TestNewIndexFile(t *testing.T) {
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewIndexFile(tt.args.block); (err != nil) != tt.wantErr {
				t.Errorf("NewIndexFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}