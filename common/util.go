package common

import (
	"encoding/binary"
	"fmt"
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
