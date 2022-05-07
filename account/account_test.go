package account

import (
	"encoding/json"
	"reflect"
	"testing"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/stretchr/testify/assert"
)

func TestJsonMarshalPermission(t *testing.T) {
	permRoot := &PermissionRoot{}
	prBytes, err := json.Marshal(permRoot)
	assert.Equal(t, nil, err)

	var pr PermissionRoot
	err = json.Unmarshal(prBytes, &pr)
	assert.Equal(t, nil, err)
	pr.IsRoot()
}

func TestJsonMarshal(t *testing.T) {
	acc := NewContractControlAccount(tpcrtypes.Address("testaddr"), "myadr", 10000)
	accBytes, err := json.Marshal(&acc)
	assert.Equal(t, nil, err)

	var accT Account
	err = json.Unmarshal(accBytes, &accT)
	assert.Equal(t, nil, err)

	assert.Equal(t, tpcrtypes.Address("testaddr"), accT.Addr)
	assert.Equal(t, AccountName("myadr"), accT.Name)
	assert.Equal(t, false, accT.Token.Permission.IsRoot())
}

func TestJsonType(t *testing.T) {
	a := 5
	b := tpcrtypes.Address("testB")

	testMap := map[string]interface{}{reflect.TypeOf(a).String(): a, reflect.TypeOf(b).String(): b}

	testBytes, err := json.Marshal(&testMap)
	assert.Equal(t, nil, err)

	var c map[string]interface{}
	err = json.Unmarshal(testBytes, &c)
	assert.Equal(t, nil, err)
}
