
package block

import (
"fmt"
"os"
"testing"
"unsafe"

//"github.com/fatih/structs"
//"reflecting"

//"fmt"
//"github.com/TopiaNetwork/topia/chain/types"
)




func TestCompress(t *testing.T) {
	testPath := "new91.txt"
	file, _ := os.OpenFile(testPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	file2, _ := os.OpenFile("new.txt", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	a := Compress(unsafe.Pointer(&file),20,unsafe.Pointer(&file2),20)
	fmt.Println(a)

}







