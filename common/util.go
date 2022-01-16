package common

import "fmt"

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args))
}

func BytesCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}
