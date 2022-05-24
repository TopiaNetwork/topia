package crypt

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

func TestSecp256Addr(t *testing.T) {
	hexPub := "02950e1cdfcb133d6024109fd489f734eeb4502418e538c28481f22bce276f248c"

	pubKey, _ := hex.DecodeString(hexPub)

	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	cytService := CreateCryptService(testLog, tpcrtypes.CryptType_Secp256)
	addr, err := cytService.CreateAddress(pubKey)
	assert.Equal(t, nil, err)

	t.Log(addr)

	contractAddr := tpcrtypes.CreateUserContractAddress(addr, 10)

	t.Log(contractAddr)

	netType, err := addr.NetworkType()
	assert.Equal(t, nil, err)
	assert.Equal(t, tpcmm.CurrentNetworkType, netType)

	cType, err := addr.CryptType()
	assert.Equal(t, nil, err)
	assert.Equal(t, tpcrtypes.CryptType_Secp256, cType)

	payload, err := addr.Payload()
	assert.Equal(t, nil, err)
	assert.Equal(t, tpcrtypes.AddressLen_Secp256, len(payload))
}

func TestNativeContractAddr(t *testing.T) {
	addr := tpcrtypes.CreateNativeContractAddress(1)
	t.Log(addr)
	addr = tpcrtypes.CreateNativeContractAddress(2)
	t.Log(addr)
	addr = tpcrtypes.CreateNativeContractAddress(3)
	t.Log(addr)
	addr = tpcrtypes.CreateNativeContractAddress(4)
	t.Log(addr)
	addr = tpcrtypes.CreateNativeContractAddress(5)
	t.Log(addr)
	addr = tpcrtypes.CreateNativeContractAddress(100)
	t.Log(addr)
}
