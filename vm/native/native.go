package native

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"go.uber.org/atomic"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnvmcontract "github.com/TopiaNetwork/topia/vm/native/contract"
	tpvmmservice "github.com/TopiaNetwork/topia/vm/service"
	tpvmtype "github.com/TopiaNetwork/topia/vm/type"
)

var (
	errorType    = reflect.TypeOf(new(error)).Elem()
	contextType  = reflect.TypeOf(new(context.Context)).Elem()
	vmResultType = reflect.TypeOf(new(tpvmtype.VMResult)).Elem()
)

const (
	MOD_NAME = "NativeVM"
)

const (
	NativeVMVersion_V1 = 1
)

var (
	AllowContractMethodParamTypes = map[reflect.Kind]bool{
		reflect.Bool:   true,
		reflect.Int:    true,
		reflect.Int8:   true,
		reflect.Int16:  true,
		reflect.Int32:  true,
		reflect.Int64:  true,
		reflect.Uint:   true,
		reflect.Uint8:  true,
		reflect.Uint16: true,
		reflect.Uint32: true,
		reflect.Uint64: true,
		reflect.Array:  true,
		reflect.Map:    true,
		reflect.Ptr:    true,
		reflect.Slice:  true,
		reflect.String: true,
		reflect.Struct: true,
	}
)

type NativeVM struct {
	log              tplog.Logger
	state            *atomic.Bool
	sync             sync.RWMutex
	contractMethotds map[tpcrtypes.Address]map[string]*nativeContractMethod //name->methods
}

func NewNativeVM() *NativeVM {
	nvm := &NativeVM{
		state:            atomic.NewBool(true),
		contractMethotds: make(map[tpcrtypes.Address]map[string]*nativeContractMethod),
	}

	nvm.init()

	return nvm
}

func (nvm *NativeVM) init() {
	nvm.registerContract(tpcrtypes.NativeContractAddr_Account, tpnvmcontract.NewContractAccount())
	nvm.registerContract("ContractTest", tpnvmcontract.NewContractTest())
}

func (nvm *NativeVM) registerContract(addr tpcrtypes.Address, contract interface{}) {
	contractMap, ok := nvm.contractMethotds[addr]
	if !ok {
		contractMap = make(map[string]*nativeContractMethod)
		nvm.contractMethotds[addr] = contractMap
	}
	contractVal := reflect.ValueOf(contract)
	for i := 0; i < contractVal.NumMethod(); i++ {
		method := contractVal.Type().Method(i)
		funcType := method.Func.Type()

		if funcType.NumIn() < 1 || funcType.In(1) != contextType {
			panic("Please make sure that the first in parameter is context type")
		}

		if funcType.NumOut() < 1 {
			panic("Please make sure there are at least one parameter and the last is error type")
		}

		nContractMethod := &nativeContractMethod{
			receiver:            contractVal,
			method:              method.Func,
			resultDataOutSIndex: -1,
		}

		if funcType.NumOut() == 1 {
			if funcType.Out(0) == errorType {
				nContractMethod.resultErrorOutIndex = 0
			} else {
				panicStr := fmt.Sprintf("Contract %s method %s return parameter hasn't no error type", addr, method.Name)
				panic(panicStr)
			}
		}

		if funcType.NumOut() >= 2 {
			if funcType.Out(funcType.NumOut()-1) == errorType {
				nContractMethod.resultDataOutSIndex = 0
				nContractMethod.resultErrorOutIndex = funcType.NumOut() - 1
			} else {
				panicStr := fmt.Sprintf("Contract %s method %s return parameters'last parameter should be error type", addr, method.Name)
				panic(panicStr)
			}
		}

		for pIn := 1; pIn < funcType.NumIn(); pIn++ {
			pInType := funcType.In(pIn)
			if pIn > 1 {
				if _, ok := AllowContractMethodParamTypes[pInType.Kind()]; !ok {
					panicStr := fmt.Sprintf("Contract %s method %s not support parameter type %s", addr, method.Name, pInType.Kind().String())
					panic(panicStr)
				}
				if pInType.Kind() == reflect.Ptr &&
					(pInType.Elem().Kind() != reflect.Slice || pInType.Elem().Kind() != reflect.Map || pInType.Kind() != reflect.Struct) {
					panicStr := fmt.Sprintf("Contract %s method %s not support parameter ptr type %s", addr, method.Name, pInType.Elem().Kind().String())
					panic(panicStr)
				}
				if pInType.Kind() == reflect.Struct {
					panicStr := fmt.Sprintf("Contract %s method %s  parameter should be ptr with struct %s", addr, method.Name)
					panic(panicStr)
				}
			}

			nContractMethod.paramTypes = append(nContractMethod.paramTypes, pInType)

		}

		contractMap[method.Name] = nContractMethod
	}
}

func (nvm *NativeVM) Version() int {
	return NativeVMVersion_V1
}

func (nvm *NativeVM) Type() tpvmtype.VMType {
	return tpvmtype.VMType_NATIVE
}

func (nvm *NativeVM) Enable() bool {
	return nvm.state.Load()
}

func (nvm *NativeVM) UpdateState(state bool) {
	nvm.state.Swap(state)
}

func (nvm *NativeVM) SetLogger(level tplogcmm.LogLevel, log tplog.Logger) {
	nvm.log = tplog.CreateModuleLogger(level, MOD_NAME, log)
}

func (nvm *NativeVM) DeployContract(ctx *tpvmmservice.VMContext) (*tpvmtype.VMResult, error) {
	panic("All native contracts needn't be deployed")
}

func (nvm *NativeVM) doExecute(ctx context.Context, nodeID string, contractAddr tpcrtypes.Address, methodName string, ncMethod *nativeContractMethod, paramIns []reflect.Value) (*tpvmtype.VMResult, error) {
	defer func() {
		if rtn := recover(); rtn != nil {
			nvm.log.Errorf("NativeVM doExecute panic: contract %s method %s, exception %s", contractAddr, methodName, rtn)
		}
	}()

	callRtns := ncMethod.method.Call(paramIns)
	if len(callRtns) != ncMethod.resultErrorOutIndex+1 {
		err := fmt.Errorf("Invalid return result for contract %s method %s: expect len %d, actual %d", contractAddr, methodName, ncMethod.resultErrorOutIndex+1, len(callRtns))
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_InvalidReturn,
			ErrMsg: err.Error(),
		}, err
	}

	if err, ok := callRtns[len(callRtns)-1].Interface().(error); ok && err != nil {
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_ExecuteErr,
			ErrMsg: err.Error(),
		}, err
	}

	if ncMethod.resultErrorOutIndex == 0 {
		return &tpvmtype.VMResult{
			Code: tpvmtype.ReturnCode_Ok,
		}, nil
	}

	if ncMethod.resultErrorOutIndex >= 1 {
		rtnValMap := make(map[string][]byte)
		for i, callRtn := range callRtns[:(len(callRtns) - 1)] {
			callRtnBytes, _ := json.Marshal(callRtn.Interface())
			rtnValMap[callRtn.Type().String()+fmt.Sprintf("_%d", i+1)] = callRtnBytes
		}
		rtnBytes, err := json.Marshal(&rtnValMap)
		if err != nil {
			return &tpvmtype.VMResult{
				Code:   tpvmtype.ReturnCode_InvalidReturn,
				ErrMsg: err.Error(),
			}, err
		}

		vmResult := &tpvmtype.VMResult{
			Code: tpvmtype.ReturnCode_Ok,
			Data: rtnBytes,
		}

		nodeEVHub := eventhub.GetEventHubManager().GetEventHub(nodeID)
		if nodeEVHub != nil {
			nodeEVHub.Trig(ctx, eventhub.EventName_ContractInvoked, vmResultType)
		}

		return vmResult, nil
	}

	err := fmt.Errorf("Execute unknown err for contract %s method %s", contractAddr, methodName)
	return &tpvmtype.VMResult{
		Code:   tpvmtype.ReturnCode_UnknownErr,
		ErrMsg: err.Error(),
	}, err
}

func (nvm *NativeVM) ExecuteContract(ctx *tpvmmservice.VMContext) (*tpvmtype.VMResult, error) {
	if !nvm.Enable() {
		err := errors.New("Native vm not enable ")
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_NotEnable,
			ErrMsg: err.Error(),
		}, err
	}

	if ctx.ContractAddr == "" || ctx.Method == "" {
		err := fmt.Errorf("Invalid vm ctx: contract %s method %s", ctx.ContractAddr, ctx.Method)
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_InvalidVMCtx,
			ErrMsg: err.Error(),
		}, err
	}

	exeCtx := context.WithValue(ctx.Context, tpvmmservice.VMCtxKey_VMServant, ctx.VMServant)

	fromAcc, err := tpvmmservice.NewContractContext(exeCtx).GetFromAccount()
	if err != nil {
		err := fmt.Errorf("Invalid vm ctx: contract %s method %s, err %v", ctx.ContractAddr, ctx.Method, err)
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_InvalidVMCtx,
			ErrMsg: err.Error(),
		}, err
	}
	permOK := false
	if fromAcc.Token != nil && fromAcc.Token.Permission != nil {
		permOK = fromAcc.Token.Permission.HasMethodPerm(ctx.ContractAddr, ctx.Method)
	}
	if !permOK {
		err = fmt.Errorf("contract %s method %s inaccessible by addr %s", ctx.ContractAddr, ctx.Method, fromAcc.Addr)
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_PermissionErr,
			ErrMsg: err.Error(),
		}, err
	}

	contractMap, ok := nvm.contractMethotds[ctx.ContractAddr]
	if !ok {
		err := fmt.Errorf("Can't find contract %s", ctx.ContractAddr)
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_ContractNotFound,
			ErrMsg: err.Error(),
		}, err
	}

	ncMethod, ok := contractMap[ctx.Method]
	if !ok {
		err := fmt.Errorf("Can't find contract %s method %s", ctx.ContractAddr, ctx.Method)
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_MethodNotFound,
			ErrMsg: err.Error(),
		}, err
	}

	paramValues, err := ctx.ParseArgs(ncMethod.paramTypes[1:])
	if err != nil {
		err = fmt.Errorf("Invalid parameters for contract %s method %s: %v", ctx.ContractAddr, ctx.Method, err)
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_MethodSignatureErr,
			ErrMsg: err.Error(),
		}, err
	}
	if len(paramValues) != len(ncMethod.paramTypes)-1 {
		err := fmt.Errorf("Invalid parameters for contract %s method %s: expect len %d, actual %d", ctx.ContractAddr, ctx.Method, len(ncMethod.paramTypes)-1, len(paramValues))
		return &tpvmtype.VMResult{
			Code:   tpvmtype.ReturnCode_InvalidParam,
			ErrMsg: err.Error(),
		}, err
	}

	var paramIns []reflect.Value
	paramIns = append(paramIns, ncMethod.receiver)
	paramIns = append(paramIns, reflect.ValueOf(exeCtx))
	paramIns = append(paramIns, paramValues...)

	return nvm.doExecute(exeCtx, ctx.NodeID, ctx.ContractAddr, ctx.Method, ncMethod, paramIns)
}
