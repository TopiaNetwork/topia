package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	//"os"
	"testing"
)


var blocknum uint64 = 123456

//func init(t *testing.T) {
//
//}

func TestNewRollback(t *testing.T) {
	rollback,_ := NewRollback(types.BlockNum(blocknum))
	fmt.Println("",rollback)


}


//func TestFileItem_AddRollback(t *testing.T) {
//	newtestfile(string(blocknum),4)
//
//
//}

func TestRemoveBlockhead(t *testing.T) {


}

func TestRemoveBlockdata(t *testing.T) {


}

func TestRemoveindex(t *testing.T) {


}

