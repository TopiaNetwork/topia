/*
go run this file, and you will get 3 sets of public keys, private keys, messages and corresponding signatures randomly generated by Ethereum
*/

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	ethSecp256 "github.com/ethereum/go-ethereum/crypto/secp256k1"
	"io"
)

const nSets = 3 // generate n sets of Ethereum data

//ethGenerateKeyPair ETH way of generating Key Pair
func ethGenerateKeyPair() (pubkey, privkey []byte) {
	key, err := ecdsa.GenerateKey(ethSecp256.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(ethSecp256.S256(), key.X, key.Y)
	privkey = make([]byte, 32)
	blob := key.D.Bytes()
	copy(privkey[32-len(blob):], blob)
	return pubkey, privkey
}

func ethPubkeyToAddress(pub []byte) [20]byte {
	return ethCommon.BytesToAddress(ethCrypto.Keccak256(pub[1:])[12:])
}

func csprngEntropy(n int) []byte {
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	return buf
}

func main() {
	for i := 0; i < nSets; i++ {
		pubETH, secETH := ethGenerateKeyPair()
		addrBytesETH := ethPubkeyToAddress(pubETH)
		msg := csprngEntropy(32)
		sig, _ := ethSecp256.Sign(msg, secETH)
		var addrETH ethCommon.Address = addrBytesETH
		fmt.Println("ETH sec:", secETH)
		fmt.Println("ETH pub:", pubETH)
		fmt.Println("msg:", msg)
		fmt.Println("ETH sig:", sig)
		fmt.Println("ETH addr:", addrETH.Hex())
		fmt.Println("")
	}
}