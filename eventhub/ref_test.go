package eventhub

import (
	"fmt"
	tptypes "github.com/TopiaNetwork/topia/chain/types"
	"reflect"
	"testing"
)

func TestRef(t *testing.T) {
	ty := reflect.TypeOf(tptypes.Block{})
	typeName := ty.Name()
	typeString := ty.String()
	kindString := ty.Kind().String()

	fmt.Printf("typeName=%s, typeString=%s, kindString=%s\n", typeName, typeString, kindString)
}
