package eventhub

import (
	"fmt"
	"reflect"
	"testing"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
)

func TestRef(t *testing.T) {
	ty := reflect.TypeOf(tpchaintypes.Block{})
	typeName := ty.Name()
	typeString := ty.String()
	kindString := ty.Kind().String()

	fmt.Printf("typeName=%s, typeString=%s, kindString=%s\n", typeName, typeString, kindString)
}
