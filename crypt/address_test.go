package crypt

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
)

func TestSecp256Addr(t *testing.T) {
	hexPub := "02950e1cdfcb133d6024109fd489f734eeb4502418e538c28481f22bce276f248c"

	pubKey, _ := hex.DecodeString(hexPub)

	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	cytService := CreateCryptService(testLog, tpcrtypes.CryptType_Secp256)
	addr, err := cytService.CreateAddress(pubKey)
	assert.Equal(t, nil, err)

	t.Log(addr)

	contractAddr := tpcrtypes.CreateContractAddress(addr, 10)

	t.Log(contractAddr)

	netType, err := addr.NetworkType()
	assert.Equal(t, nil, err)
	assert.Equal(t, tpnet.CurrentNetworkType, netType)

	cType, err := addr.CryptType()
	assert.Equal(t, nil, err)
	assert.Equal(t, tpcrtypes.CryptType_Secp256, cType)

	payload, err := addr.Payload()
	assert.Equal(t, nil, err)
	assert.Equal(t, tpcrtypes.AddressLen_Secp256, len(payload))
}
