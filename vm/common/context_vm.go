package common

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"reflect"
	"strconv"
	"strings"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

const (
	ContractMethodArgs_Separation = "@"
)

type VMCtxKey int

const (
	VMCtxKey_Unknown VMCtxKey = iota
	VMCtxKey_VMServant
	VMCtxKey_FromAddr
)

type VMContext struct {
	context.Context
	VMServant
	NodeID       string
	ContractAddr tpcrtypes.Address
	Code         []byte
	Method       string
	Args         string
}

func (vmctx *VMContext) ParseArgs(paramTypes []reflect.Type) ([]reflect.Value, error) {
	args := strings.Split(vmctx.Args, ContractMethodArgs_Separation)
	if len(args) != len(paramTypes) {
		return nil, fmt.Errorf("Invalid args: len %d, expected %d", len(args), len(paramTypes))
	}

	var argValues []reflect.Value
	for i, paramType := range paramTypes {
		argValue := reflect.New(paramType)
		paramTypeKind := paramType.Kind()
		if paramTypeKind != reflect.Slice && paramTypeKind != reflect.Map && paramType.Kind() != reflect.Ptr {
			argValue = reflect.New(paramType)
			if paramTypeKind == reflect.Int || paramTypeKind == reflect.Int8 ||
				paramTypeKind == reflect.Int16 || paramTypeKind == reflect.Int32 || paramTypeKind == reflect.Int64 {
				pVal, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					return nil, err
				}
				argValue.SetInt(pVal)
			}

			if paramTypeKind == reflect.Uint || paramTypeKind == reflect.Uint8 ||
				paramTypeKind == reflect.Uint16 || paramTypeKind == reflect.Uint32 || paramTypeKind == reflect.Uint64 {
				pVal, err := strconv.ParseUint(args[i], 10, 64)
				if err != nil {
					return nil, err
				}
				argValue.SetUint(pVal)
			}

			if paramTypeKind == reflect.String {
				argValue.Elem().SetString(args[i])
			}

			if paramTypeKind == reflect.Bool {
				pVal, err := strconv.ParseBool(args[i])
				if err != nil {
					return nil, err
				}
				argValue.SetBool(pVal)
			}

			argValues = append(argValues, argValue.Elem())
		} else {
			if paramTypeKind == reflect.Ptr {
				argValue = reflect.New(paramType.Elem())
			}

			paramBytes, err := hex.DecodeString(args[i])
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(paramBytes, argValue)
			if err != nil {
				return nil, err
			}

			if paramTypeKind == reflect.Ptr {
				argValues = append(argValues, argValue)
			} else {
				argValues = append(argValues, argValue.Elem())
			}
		}
	}

	return argValues, nil
}

func (vkey VMCtxKey) String() string {
	switch vkey {
	case VMCtxKey_VMServant:
		return "VMServant"
	case VMCtxKey_FromAddr:
		return "FromAddr"
	default:
		return "unknown"
	}
}

func (vkey VMCtxKey) Value(vkeyStr string) VMCtxKey {
	switch vkeyStr {
	case "VMServant":
		return VMCtxKey_VMServant
	case "FromAddr":
		return VMCtxKey_FromAddr
	default:
		return VMCtxKey_Unknown
	}
}
