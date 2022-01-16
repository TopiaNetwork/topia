package state

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/lazyledger/smt"
)

func TestBasicSMT(t *testing.T) {
	store := smt.NewSimpleMap()
	tree := smt.NewSparseMerkleTree(store, store, sha256.New())

	tree.Update([]byte("foo"), []byte("bar"))

	proof, _ := tree.Prove([]byte("foo"))
	root := tree.Root()

	if smt.VerifyProof(proof, root, []byte("foo"), []byte("bar"), sha256.New()) {
		fmt.Println("Proof verification succeeded.")
	} else {
		fmt.Println("Proof verification failed.")
	}

	val, _ := tree.Get([]byte("foo"))
	fmt.Printf("val:%s\n", string(val))
}
