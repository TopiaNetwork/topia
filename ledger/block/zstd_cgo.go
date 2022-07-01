package block


//#cgo CFLAGS: -I./zstd
//#cgo LDFLAGS: -L${SRCDIR}/zstd -lzstd
//
//#include "zstd.h"
import "C"
import "unsafe"

const DefaultCompressionLevel = 3

func Compress(dst unsafe.Pointer, dstCapacity uint32, src unsafe.Pointer, srcSize uint32)  uint32{
	res := C.ZSTD_compress(dst, C.ulong(dstCapacity),src, C.ulong(srcSize), C.int(DefaultCompressionLevel))
	return is_error(uint32(res))
}


func Decompress(dst unsafe.Pointer, dstCapacity uint32, src unsafe.Pointer, srcSize uint32)  uint32{
	res := C.ZSTD_decompress(dst, C.ulong(dstCapacity),src, C.ulong(srcSize))
	return is_error(uint32(res))
}


func is_error(code uint32) uint32{

	return uint32(C.ZSTD_isError(C.ulong(code)))
}