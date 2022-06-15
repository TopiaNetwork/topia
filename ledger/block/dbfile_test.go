package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"os"
	"testing"
	//"reflecting"

	//"fmt"
	//"github.com/TopiaNetwork/topia/chain/types"
)

var blockhead1 = types.BlockHead{
	nil,
	5,
	1,
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
	1,
	nil,
	nil,
	nil,
	nil,
	0,0,nil,nil,struct{}{} ,nil,0,
}
var blockdata = types.BlockData{
	1,
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
	1,
}
//struct block1{
//
//}



func Test_outSize(t *testing.T) {

	b := outSize("text.txt")
	fmt.Printf("%t",b)


}

func TestNewFile(t *testing.T) {
	topia,_ := NewFile(&block_all,0)
	fmt.Printf("",topia)

	index,_ := NewFile(&block_all,1)
	fmt.Printf("",index)

	trans,_ := NewFile(&block_all,1)
	fmt.Printf("",trans)
}

func TestTopiaFile_Writedata(t *testing.T) {
	file, err := os.OpenFile("test.topia", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = TopiaFile{
		1,
		file,
		1,

	}
	err =to.Writedata(&block_all)
	if err != nil{
		panic(err)
	}
}


func TestTopiaFile_Writeindex(t *testing.T) {
	file, err := os.OpenFile("test.index", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = TopiaFile{
		1,
		file,
		1,

	}
	err =to.Writedata(&block_all)
	if err != nil{
		panic(err)
	}
}


func TestTopiaFile_Writetrans(t *testing.T) {
	file, err := os.OpenFile("test.trans", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = TopiaFile{
		1,
		file,
		1,

	}
	err =to.Writedata(&block_all)
	if err != nil{
		panic(err)
	}
}

func TestTopiaFile_FindBlock(t *testing.T) {
	
}

//func TestnewDataFile(t *testing.T) {
//
//}





