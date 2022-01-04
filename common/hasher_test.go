package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/blake2b"
	"testing"
)

func TestBlake2bHasherWithSize(t *testing.T) {
	h := NewBlake2bHasher(64)
	require.NotEqual(t, nil, h)
	require.Equal(t, 64, h.Size())

	hashBytes1 := h.Compute("")
	require.NotEqual(t, nil, hashBytes1)
	t.Logf("len=%d, hashBytes=%v", len(hashBytes1), fmt.Sprintf("%x", hashBytes1))

	hashBytes2 := h.Compute("teststring")
	require.NotEqual(t, nil, hashBytes2)
	t.Logf("len=%d, hashBytes=%v", len(hashBytes2), fmt.Sprintf("%x", hashBytes2))
}

func TestBlake2bHasherWithoutSize(t *testing.T) {
	h := NewBlake2bHasher(0)
	require.NotEqual(t, nil, h)
	require.Equal(t, blake2b.Size256, h.Size())

	hashBytes1 := h.Compute("")
	require.NotEqual(t, nil, hashBytes1)
	t.Logf("len=%d, hashBytes=%v", len(hashBytes1), fmt.Sprintf("%x", hashBytes1))

	hashBytes2 := h.Compute("teststring")
	require.NotEqual(t, nil, hashBytes2)
	t.Logf("len=%d, hashBytes=%v", len(hashBytes2), fmt.Sprintf("%x", hashBytes2))
}
