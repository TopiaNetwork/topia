package common

import (
	"encoding/binary"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"sort"
)

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args))
}

func BytesCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}

func Uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b[:]
}

func BytesToUint64(d []byte) uint64 {
	return binary.BigEndian.Uint64(d)
}

func IsContainString(target string, str_array []string) bool {
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	//index：0 ~ (len(str_array)-1), The return value is 1 if target is not present
	return index < len(str_array) && str_array[index] == target
}

func IsContainItem(target interface{}, array []interface{}) bool {
	return mapset.NewSetFromSlice(array).Contains(target)
}

func Has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func IsHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

func IsHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !IsHexCharacter(c) {
			return false
		}
	}
	return true
}
