package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
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
//struct block1{
//
//}



func Test_outSize(t *testing.T) {

	b := outSize("text.txt")
	fmt.Printf("%t",b)


}

func TestNewFile(t *testing.T) {

}

func TestTopiaFile_Writedata(t *testing.T) {

}


func TestTopiaFile_Writeindex(t *testing.T) {

}




//func TestnewDataFile(t *testing.T) {
//
//}





