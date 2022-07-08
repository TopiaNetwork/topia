package block

import(
	"bytes"
	//"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"math"
	"testing"
)


func TestNew(t *testing.T) {

}

func Testsum128(t *testing.T){
	//pad_tail := true
	//length := 20
	//tail := make([]byte,10)

	//h1,h2 := sum128(pad_tail,length,tail)

}




func TestNewWithLowNumbers(t *testing.T) {
	f := New()
	if f.k != 1 {
		t.Errorf("%v should be 1", f.k)
	}
	if f.m != 1 {
		t.Errorf("%v should be 1", f.m)
	}
}


func min(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}


func chiTestBloom(m, k, rounds uint, elements [][]byte) (succeeds bool) {
	f := New()
	results := make([]uint, m)
	chi := make([]float64, m)

	for _, data := range elements {
		h := baseHashes(data)
		for i := uint(0); i < f.k; i++ {
			results[f.location(h, i)]++
		}
	}

	var chiStatistic float64
	e := float64(k*rounds) / float64(m)
	for i := uint(0); i < m; i++ {
		chi[i] = math.Pow(float64(results[i])-e, 2.0) / e
		chiStatistic += chi[i]
	}

	table := [20]float64{
		7.879, 10.597, 12.838, 14.86, 16.75, 18.548, 20.278,
		21.955, 23.589, 25.188, 26.757, 28.3, 29.819, 31.319, 32.801, 34.267,
		35.718, 37.156, 38.582, 39.997}
	df := min(m-1, 20)

	succeeds = table[df-1] > chiStatistic
	return

}

func TestLocation(t *testing.T) {
	var m, k, rounds uint

	m = 8
	k = 3

	rounds = 100000 // 15000000

	elements := make([][]byte, rounds)

	for x := uint(0); x < rounds; x++ {
		ctrlist := make([]uint8, 4)
		ctrlist[0] = uint8(x)
		ctrlist[1] = uint8(x >> 8)
		ctrlist[2] = uint8(x >> 16)
		ctrlist[3] = uint8(x >> 24)
		data := []byte(ctrlist)
		elements[x] = data
	}

	succeeds := chiTestBloom(m, k, rounds, elements)
	if !succeeds {
		t.Error("random assignment is too unrandom")
	}

}

func TestCap(t *testing.T) {
	f := New()
	if f.Cap() != f.m {
		t.Error("not accessing Cap() correctly")
	}
}


func TestMarshalUnmarshalJSON(t *testing.T) {
	f := New()
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err.Error())
	}

	var g BloomFilter
	err = json.Unmarshal(data, &g)
	if err != nil {
		t.Fatal(err.Error())
	}
	if g.m != f.m {
		t.Error("invalid m value")
	}
	if g.k != f.k {
		t.Error("invalid k value")
	}
	if g.b == nil {
		t.Fatal("bitset is nil")
	}
	if !g.b.Equal(f.b) {
		t.Error("bitsets are not equal")
	}
}



func TestEncodeDecodeGob(t *testing.T) {
	f := New()
	f.Add([]byte("one"))
	f.Add([]byte("two"))
	f.Add([]byte("three"))
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(f)
	if err != nil {
		t.Fatal(err.Error())
	}

	var g BloomFilter
	err = gob.NewDecoder(&buf).Decode(&g)
	if err != nil {
		t.Fatal(err.Error())
	}
	if g.m != f.m {
		t.Error("invalid m value")
	}
	if g.k != f.k {
		t.Error("invalid k value")
	}
	if g.b == nil {
		t.Fatal("bitset is nil")
	}
	if !g.b.Equal(f.b) {
		t.Error("bitsets are not equal")
	}
	if !g.Find([]byte("three")) {
		t.Errorf("missing value 'three'")
	}
	if !g.Find([]byte("two")) {
		t.Errorf("missing value 'two'")
	}
	if !g.Find([]byte("one")) {
		t.Errorf("missing value 'one'")
	}
}

func TestEqual(t *testing.T) {
	f := New()
	f1 := New()
	g := New()
	h := New()
	n1 := []byte("Bess")
	f1.Add(n1)
	if !f.Equal(f) {
		t.Errorf("%v should be equal to itself", f)
	}
	if f.Equal(f1) {
		t.Errorf("%v should not be equal to %v", f, f1)
	}
	if f.Equal(g) {
		t.Errorf("%v should not be equal to %v", f, g)
	}
	if f.Equal(h) {
		t.Errorf("%v should not be equal to %v", f, h)
	}
}



func TestFrom(t *testing.T) {
	var (
		k    = uint(5)
		data = make([]uint64, 10)
		test = []byte("test")
	)

	bf := From(data, k)

	if bf.Cap() != uint(len(data)*64) {
		t.Errorf("Capacity does not match the expected value")
	}

	if bf.Find(test) {
		t.Errorf("Bloom filter should not contain the value")
	}

	bf.Add(test)
	if !bf.Find(test) {
		t.Errorf("Bloom filter should contain the value")
	}

	// create a new Bloom filter from an existing (populated) data slice.
	bf = From(data, k)
	if !bf.Find(test) {
		t.Errorf("Bloom filter should contain the value")
	}
}

