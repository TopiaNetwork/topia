package block

import (
	"github.com/bits-and-blooms/bitset"
	"math/bits"
	"unsafe"
)



type BloomFilter struct {
	m uint
	k uint
	b *bitset.BitSet
}

func max(x, y uint) uint {
	if x > y {
		return x
	}
	return y
}

type digest128 struct {
	h1 uint64 // Unfinalized running hash part 1.
	h2 uint64 // Unfinalized running hash part 2.
}

//m = uint(math.Ceil(-1 * float64(n) * math.Log(p) / math.Pow(math.Log(2), 2)))
//k = uint(math.Ceil(math.Log(2) * float64(m) / float64(n)))

const (
	c1_128     = 0x87c37b91114253d5
	c2_128     = 0x4cf5ad432745937f
	block_size = 16

	count = 100000000
	m = 4313276270
	p = 0.000000001
	k = 30 //29.93
)

func New() *BloomFilter {
	return &BloomFilter{max(1, m), max(1, k), bitset.New(m)}
}


func From(data []uint64, k uint) *BloomFilter {
	m := uint(len(data) * 64)
	return FromWithM(data, m, k)
}


func FromWithM(data []uint64, m, k uint) *BloomFilter {
	return &BloomFilter{m, k, bitset.From(data)}
}


func baseHashes(data []byte) [4]uint64 {
	var d digest128 // murmur hashing
	hash1, hash2, hash3, hash4 := d.sum256(data)
	return [4]uint64{
		hash1, hash2, hash3, hash4,
	}
}

func (f *BloomFilter) Add(data []byte) *BloomFilter {
	h := baseHashes(data)
	for i := uint(0); i < f.k; i++ {
		f.b.Set(f.location(h, i))
	}
	return f
}

func location(h [4]uint64, i uint) uint64 {
	ii := uint64(i)
	return h[ii%2] + ii*h[2+(((ii+(ii%2))%4)/2)]
}

func (f *BloomFilter) location(h [4]uint64, i uint) uint {
	return uint(location(h, i) % uint64(f.m))
}

func (f *BloomFilter) Find(data []byte) bool {
	h := baseHashes(data)
	for i := uint(0); i < f.k; i++ {
		if !f.b.Test(f.location(h, i)) {
			return false
		}
	}
	return true
}
func (f *BloomFilter) Cap() uint {
	return f.m
}


func (f *BloomFilter) BitSet() *bitset.BitSet {
	return f.b
}


func (d *digest128) sum256(data []byte) (hash1, hash2, hash3, hash4 uint64) {

	d.h1, d.h2 = 0, 0

	d.bmix(data)

	length := uint(len(data))
	tail_length := length % block_size
	tail := data[length-tail_length:]
	hash1, hash2 = d.sum128(false, length, tail)

	if tail_length+1 == block_size {

		word1 := *(*uint64)(unsafe.Pointer(&tail[0]))
		word2 := uint64(*(*uint32)(unsafe.Pointer(&tail[8])))
		word2 = word2 | (uint64(tail[12]) << 32) | (uint64(tail[13]) << 40) | (uint64(tail[14]) << 48)

		word2 = word2 | (uint64(1) << 56)

		d.bmix_words(word1, word2)
		tail := data[length:]
		hash3, hash4 = d.sum128(false, length+1, tail)
	} else {

		hash3, hash4 = d.sum128(true, length+1, tail)
	}

	return hash1, hash2, hash3, hash4
}

func (f *BloomFilter) Equal(g *BloomFilter) bool {
	return f.m == g.m && f.k == g.k && f.b.Equal(g.b)
}


func Locations(data []byte, k uint) []uint64 {
	locs := make([]uint64, k)


	h := baseHashes(data)
	for i := uint(0); i < k; i++ {
		locs[i] = location(h, i)
	}

	return locs
}

func (d *digest128) sum128(pad_tail bool, length uint, tail []byte) (h1, h2 uint64) {
	h1, h2 = d.h1, d.h2

	var k1, k2 uint64
	if pad_tail {
		switch (len(tail) + 1) & 15 {
		case 15:
			k2 ^= uint64(1) << 48
			break
		case 14:
			k2 ^= uint64(1) << 40
			break
		case 13:
			k2 ^= uint64(1) << 32
			break
		case 12:
			k2 ^= uint64(1) << 24
			break
		case 11:
			k2 ^= uint64(1) << 16
			break
		case 10:
			k2 ^= uint64(1) << 8
			break
		case 9:
			k2 ^= uint64(1) << 0

			k2 *= c2_128
			k2 = bits.RotateLeft64(k2, 33)
			k2 *= c1_128
			h2 ^= k2

			break

		case 8:
			k1 ^= uint64(1) << 56
			break
		case 7:
			k1 ^= uint64(1) << 48
			break
		case 6:
			k1 ^= uint64(1) << 40
			break
		case 5:
			k1 ^= uint64(1) << 32
			break
		case 4:
			k1 ^= uint64(1) << 24
			break
		case 3:
			k1 ^= uint64(1) << 16
			break
		case 2:
			k1 ^= uint64(1) << 8
			break
		case 1:
			k1 ^= uint64(1) << 0
			k1 *= c1_128
			k1 = bits.RotateLeft64(k1, 31)
			k1 *= c2_128
			h1 ^= k1
		}

	}
	switch len(tail) & 15 {
	case 15:
		k2 ^= uint64(tail[14]) << 48
		fallthrough
	case 14:
		k2 ^= uint64(tail[13]) << 40
		fallthrough
	case 13:
		k2 ^= uint64(tail[12]) << 32
		fallthrough
	case 12:
		k2 ^= uint64(tail[11]) << 24
		fallthrough
	case 11:
		k2 ^= uint64(tail[10]) << 16
		fallthrough
	case 10:
		k2 ^= uint64(tail[9]) << 8
		fallthrough
	case 9:
		k2 ^= uint64(tail[8]) << 0

		k2 *= c2_128
		k2 = bits.RotateLeft64(k2, 33)
		k2 *= c1_128
		h2 ^= k2

		fallthrough

	case 8:
		k1 ^= uint64(tail[7]) << 56
		fallthrough
	case 7:
		k1 ^= uint64(tail[6]) << 48
		fallthrough
	case 6:
		k1 ^= uint64(tail[5]) << 40
		fallthrough
	case 5:
		k1 ^= uint64(tail[4]) << 32
		fallthrough
	case 4:
		k1 ^= uint64(tail[3]) << 24
		fallthrough
	case 3:
		k1 ^= uint64(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint64(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint64(tail[0]) << 0
		k1 *= c1_128
		k1 = bits.RotateLeft64(k1, 31)
		k1 *= c2_128
		h1 ^= k1
	}

	h1 ^= uint64(length)
	h2 ^= uint64(length)

	h1 += h2
	h2 += h1

	h1 = fmix64(h1)
	h2 = fmix64(h2)

	h1 += h2
	h2 += h1

	return h1, h2
}

func fmix64(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33
	return k
}

func (d *digest128) bmix_words(k1, k2 uint64) {
	h1, h2 := d.h1, d.h2

	k1 *= c1_128
	k1 = bits.RotateLeft64(k1, 31)
	k1 *= c2_128
	h1 ^= k1

	h1 = bits.RotateLeft64(h1, 27)
	h1 += h2
	h1 = h1*5 + 0x52dce729

	k2 *= c2_128
	k2 = bits.RotateLeft64(k2, 33)
	k2 *= c1_128
	h2 ^= k2

	h2 = bits.RotateLeft64(h2, 31)
	h2 += h1
	h2 = h2*5 + 0x38495ab5
	d.h1, d.h2 = h1, h2
}

func (d *digest128) bmix(p []byte) {
	nblocks := len(p) / block_size
	for i := 0; i < nblocks; i++ {
		t := (*[2]uint64)(unsafe.Pointer(&p[i*block_size]))
		k1, k2 := t[0], t[1]
		d.bmix_words(k1, k2)
	}
}