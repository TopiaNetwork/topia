package rpc

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/sony/sonyflake"
)

// IsPublic is public
func IsPublic(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

func IsPublicOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return IsPublic(t.Name()) || t.PkgPath() == ""
}

// create interface{} by refType
func RefNew(refType reflect.Type) interface{} {
	var refValue reflect.Value
	if refType.Kind() == reflect.Ptr {
		refValue = reflect.New(refType.Elem())
	} else {
		refValue = reflect.New(refType)
	}

	return refValue.Interface()
}

func PrintStack() {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	fmt.Printf("==> %s\n", string(buf[:n]))
}

var (
	sonyFlake *sonyflake.Sonyflake
)

func init() {
	st := sonyflake.Settings{}
	sonyFlake = sonyflake.NewSonyflake(st)
}

func DistributedID() (string, error) {
	id, err := sonyFlake.NextID()
	if err != nil {
		return "", err
	}

	return strconv.Itoa(int(id)), nil
}

func Encode(data interface{}) ([]byte, error) {
	buf, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func Decode(b []byte) (data []interface{}, err error) {
	err = json.Unmarshal(b, &data)
	if err != nil {
		return make([]interface{}, 0), err
	}
	return data, nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
