package block


import (
	"fmt"
	//"os"
	"testing"
	"github.com/TopiaNetwork/topia/chain/types"
)


var blocknum uint64 = 123456


func TestNewRollback(t *testing.T) {
	rollback,_ := NewRollback(types.BlockNum(blocknum))
	fmt.Println("",rollback)


}


func TestFileItem_AddRollback(t *testing.T) {

}

func TestRemoveBlockhead(t *testing.T) {

}

func TestRemoveBlockdata(t *testing.T) {

}

func TestRemoveindex(t *testing.T) {

}

