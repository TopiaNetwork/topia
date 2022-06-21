package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"os"
	"testing"
	//"github.com/fatih/structs"
	//"reflecting"

	//"fmt"
	//"github.com/TopiaNetwork/topia/chain/types"
)

var blockhead1 = types.BlockHead{
	nil,
	5,
	1817128,
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
var blockdata = types.BlockData{
	6,
	nil,
	struct{}{},
	nil,
	1,
}

var block_all = types.Block{
	&blockhead1,
	&blockdata,
	struct{}{},
	nil,
	100,
}

var FilenameNow = "1817128"
//struct block1{
//
//}



func TestOutSize(t *testing.T) {

	b := OutSize("text.txt")
	fmt.Printf("%t",b)


}

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

func TestNewFile(t *testing.T) {
	topia,_ := NewFile(&block_all)
	fmt.Println("",topia)

	//index,_ := NewFile(&block_all,1)
	//fmt.Println("",index)
	//
	//trans,_ := NewFile(&block_all,2)
	//fmt.Println("",trans)
}

//
//func TestReaddata(t *testing.T) {
//	filename := FilenameNow + ".topia"
//	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//
//	if err != nil{
//		panic(err)
//	}
//
//	var to = TopiaFile{
//		1,
//		file,
//		1,
//
//	}
//	err =to.Writedata(&block_all)
//	if err != nil{
//		panic(err)
//	}
//}

//
//func TestWritedata(t *testing.T) {
//	file, err := os.OpenFile("test.topia", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//
//	if err != nil{
//		panic(err)
//	}
//
//	var to = TopiaFile{
//		1,
//		file,
//		1,
//
//	}
//	err =to.Writedata(&block_all)
//	if err != nil{
//		panic(err)
//	}
//}
//
//
func TestTopiaFile_Findindex(t *testing.T) {
	filename := FilenameNow + ".index"
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = FileItem{
		1,
		file,
		0,

	}
	tpindex,_ :=to.Findindex(1817128)
	//if err != nil{
	//	panic(err)
	//}
	fmt.Println(tpindex)
}

//func TestWriteindex(t *testing.T) {
//	file, err := os.OpenFile("test.index", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
//
//	if err != nil{
//		panic(err)
//	}
//
//	var to = FileItem{
//		1,
//		file,
//		1,
//
//	}
//	err =to.Writedata(&block_all)
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
//	var to = TopiaFile{
//		1,
//		file,
//		1,
//
//	}
//	err =to.Writedata(&block_all)
//	if err != nil{
//		panic(err)
//	}
//}
//
func TestFindBlock(t *testing.T) {

	//k := reflect.TypeOf(blockhead1)
	//v := reflect.ValueOf(blockhead1)
	//
	//fmt.Println("",k)
	//fmt.Println("",v)

	filename := FilenameNow + ".topia"
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = FileItem{
		0,
		file,
		0,

	}
	tpdata,_ :=to.FindBlockbyNumber(1817128)
	//if err != nil{
	//	panic(err)
	//}
	fmt.Println(tpdata)
}



//func TestnewDataFile(t *testing.T) {
//
//}





