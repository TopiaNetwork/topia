package block


/*

#cgo LDFLAGS: -lstdc++

#include "./zstd/lib/common/mem.h"
#include "./zstd/lib/compress/hist.c"
#include "./zstd/lib/common/fse.h"
#include "./zstd/lib/common/huf.h"
#include "./zstd/lib/common/error_private.h"
#include "./zstd/lib/common/error_private.c"
#include "./zstd/lib/common/entropy_common.c"
#include "./zstd/lib/common/zstd_common.c"
#include "./zstd/lib/common/zstd_deps.h"
#include "./zstd/lib/common/zstd_internal.h"
#include "./zstd/lib/common/zstd_trace.h"
#include "./zstd/lib/common/fse_decompress.c"
#include  "./zstd/lib/common/bits.h"
#include  "./zstd/lib/common/xxhash.h"
#include  "./zstd/lib/common/xxhash.c"

#include "./zstd/lib/compress/zstd_compress_internal.h"
#include "./zstd/lib/compress/zstd_compress_sequences.c"
#include "./zstd/lib/compress/zstd_compress_literals.c"
#include "./zstd/lib/compress/zstd_fast.c"
#include "./zstd/lib/compress/zstd_double_fast.c"
#include "./zstd/lib/compress/zstd_lazy.c"
#include "./zstd/lib/compress/huf_compress.c"
#include "./zstd/lib/compress/zstd_opt.c"
#include "./zstd/lib/compress/zstd_ldm.c"
#include "./zstd/lib/compress/zstd_ldm.h"
#include "./zstd/lib/compress/fse_compress.c"
#include "./zstd/lib/compress/zstd_compress_superblock.h"
#include "./zstd/lib/compress/zstd_compress_superblock.c"

#include "./zstd/lib/zstd.h"
#include "./zstd/lib/compress/zstd_compress.c"
*/
import "C"
import "unsafe"

const DefaultCompressionLevel = 3

func Compress(dst unsafe.Pointer, dstCapacity uint32, src unsafe.Pointer, srcSize uint32)  uint32{
	return C.ZSTD_compress(dst, dstCapacity,src, srcSize, DefaultCompressionLevel)
}


//func Decompress(dst, src []byte) []byte {
//	return compressDictLevel(dst, src, nil, DefaultCompressionLevel)
//}