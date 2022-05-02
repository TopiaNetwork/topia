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

type VMContext struct {
	context.Context
	VMServant
	NodeID       string
	ContractAddr tpcrtypes.Address
	Code         []byte
	Method       string
	Args         string
}

type VMResult struct {
	Code    ReturnCode
	ErrMsg  string
	GasUsed uint64
	Data    interface{}
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
		if paramTypeKind != reflect.Slice && paramTypeKind == reflect.Map && paramType.Kind() == reflect.Ptr {
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

			if paramTypeKind == reflect.Bool {
				pVal, err := strconv.ParseBool(args[i])
				if err != nil {
					return nil, err
				}
				argValue.SetBool(pVal)
			}
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
		}

		argValues = append(argValues, argValue)
	}

	return argValues, nil
}
