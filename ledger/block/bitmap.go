package block

import (
	"bytes"
	"fmt"
	"github.com/RoaringBitmap/roaring"
	"github.com/TopiaNetwork/topia/chain/types"
)


func exist(blockNum types.BlockNum) {


	rb1 := roaring.BitmapOf(uint32(blockNum))
	fmt.Println(rb1.String())



	i := rb1.Iterator()
	for i.HasNext() {
		fmt.Println(i.Next())
	}
	fmt.Println()


	buf := new(bytes.Buffer)
	rb1.WriteTo(buf) // we omit error handling
	newrb:= roaring.New()
	newrb.ReadFrom(buf)
	if rb1.Equals(newrb) {
		fmt.Println("")
	}

}