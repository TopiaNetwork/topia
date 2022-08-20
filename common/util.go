package common

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math/rand"
	"sort"
	"time"

	mapset "github.com/deckarep/golang-set"
)

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func Clone(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func BytesCopy(src []byte) []byte {
	if src == nil {
		return nil
	}

	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}

func DeepCopyByGob(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func CloneSlice(a [][]byte) [][]byte {
	other := make([][]byte, len(a))
	for i := range a {
		other[i] = BytesCopy(a[i])
	}
	return other
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

func RemoveIfExistString(target string, str_array []string) []string {
	var rtnArray []string
	for i := 0; i < len(str_array); i++ {
		if str_array[i] != target {
			rtnArray = append(rtnArray, str_array[i])
		}
	}

	return rtnArray
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

//Generate count randon number in [start, end)
func GenerateRandomNumber(start int, end int, count int) []int {
	if end < start || (end-start) < count {
		return nil
	}

	nums := make([]int, 0)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for len(nums) < count {
		num := r.Intn((end - start)) + start

		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}

		if !exist {
			nums = append(nums, num)
		}
	}

	return nums
}

func MinUint64(x, y uint64) uint64 {
	if x > y {
		return y
	}
	return x
}
func MinInt64(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}
func NumberToByte(number interface{}) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, number)
	return buf.Bytes()
}

func BytesToNumber(ptr interface{}, b []byte) {
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, ptr)
}
