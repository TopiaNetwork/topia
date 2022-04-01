package common

import "testing"

func TestClone(t *testing.T) {
	b := []byte{0x01, 0x02, 0x05, 0x05, 0x07}
	a := new([]byte)

	err := Clone(a, b)

	t.Logf("%v, %v", a, err)
}
