package consensus

import (
	"fmt"
	"testing"

	"lukechampine.com/frand"
)

func TestBasicFrandShuffle(t *testing.T) {
	records := []string{"a", "b", "c", "e", "f"}

	fmt.Println(records)

	rng := frand.NewCustom([]byte("7c85524a3202e3a2f135262c3f403982"), 1024, 12)

	rng.Shuffle(len(records), func(i, j int) {
		records[i], records[j] = records[j], records[i]
	})

	fmt.Println(records)
}
