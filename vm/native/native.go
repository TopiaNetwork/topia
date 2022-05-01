package native

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"go.uber.org/atomic"

	"github.com/TopiaNetwork/topia/eventhub"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
	tpnvmcontract "github.com/TopiaNetwork/topia/vm/native/contract"
)

var (
	errorType    = reflect.TypeOf(new(error)).Elem()
	contextType  = reflect.TypeOf(new(context.Context)).Elem()
	vmResultType = reflect.TypeOf(new(tpvmcmm.VMResult)).Elem()
)

const (
	MOD_NAME = "NativeVM"
)

const (
	NativeVMVersion_V1 = 1
)

type NativeVM struct {
	log              tplog.Logger
	state            *atomic.Bool
	sync             sync.RWMutex
	contractMethotds map[string]map[string]*nativeContractMethod //name->methods
}

func NewNativeVM() *NativeVM {
	nvm := &NativeVM{
		state:            atomic.NewBool(true),
		contractMethotds: make(map[string]map[string]*nativeContractMethod),
	}

	nvm.init()

	return nvm
}

func (nvm *NativeVM) init() {
	nvm.registerContract("ContractTest", tpnvmcontract.NewContractTest())
}

func (nvm *NativeVM) registerContract(name string, contract interface{}) {
	contractMap, ok := nvm.contractMethotds[name]
	if !ok {
		contractMap = make(map[string]*nativeContractMethod)
		nvm.contractMethotds[name] = contractMap
	}
	contractVal := reflect.ValueOf(contract)
	for i := 0; i < contractVal.NumMethod(); i++ {
		method := contractVal.Type().Method(i)
		funcType := method.Func.Type()

		if funcType.NumIn() < 1 || funcType.In(1) != contextType {
			panic("Please make sure that the first in parameter is context type")
		}

		if funcType.NumOut() > 2 || funcType.NumOut() < 1 {
			panic("Please make sure there are at most two parameters and the last is error type")
		}

		nContractMethod := &nativeContractMethod{
			receiver: contractVal,
			method:   method.Func,
		}

		if funcType.NumOut() == 1 {
			if funcType.Out(0) == errorType {
				nContractMethod.resultErrorOutIndex = 0
			} else {
				panicStr := fmt.Sprintf("Contract %s method %s return parameter hasn't no error type", name, method.Name)
				panic(panicStr)
			}
		}

		if funcType.NumOut() == 2 {
			if funcType.Out(1) == errorType {
				nContractMethod.resultDataOutIndex = 0
				nContractMethod.resultErrorOutIndex = 1
			} else {
				panicStr := fmt.Sprintf("Contract %s method %s return parameters'last parameter should be error type", name, method.Name)
				panic(panicStr)
			}
		}

		for pIn := 1; pIn < funcType.NumIn(); pIn++ {
			nContractMethod.paramTypes = append(nContractMethod.paramTypes, funcType.In(pIn))
		}

		contractMap[method.Name] = nContractMethod
	}
}

func (nvm *NativeVM) Version() int {
	return NativeVMVersion_V1
}

func (nvm *NativeVM) Type() tpvmcmm.VMType {
	return tpvmcmm.VMType_NATIVE
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

func (nvm *NativeVM) DeployContract(ctx *tpvmcmm.VMContext) (*tpvmcmm.VMResult, error) {
	panic("All native contracts needn't be deployed")
}

func (nvm *NativeVM) doExecute(ctx context.Context, nodeID string, contractName string, methodName string, ncMethod *nativeContractMethod, paramIns []reflect.Value) (*tpvmcmm.VMResult, error) {
	defer func() {
		if rtn := recover(); rtn != nil {
			nvm.log.Errorf("NativeVM doExecute panic: contract %s method %s, exception %s", contractName, methodName, rtn)
		}
	}()

	callRtn := ncMethod.method.Call(paramIns)
	if len(callRtn) != 2 && ncMethod.resultErrorOutIndex == 1 || (len(callRtn) != 1 && ncMethod.resultErrorOutIndex == 0) {
		expectedLen := 1
		if ncMethod.resultErrorOutIndex == 1 {
			expectedLen = 2
		}

		err := fmt.Errorf("Invalid return result for contract %s method %s: expect len %d, actual %d", contractName, methodName, expectedLen, len(callRtn))
		return &tpvmcmm.VMResult{
			Code:   tpvmcmm.ReturnCode_InvalidReturn,
			ErrMsg: err.Error(),
		}, err
	}

	if ncMethod.resultErrorOutIndex == 0 {
		err := callRtn[0].Interface().(error)
		if err != nil {
			return &tpvmcmm.VMResult{
				Code:   tpvmcmm.ReturnCode_MethodErr,
				ErrMsg: err.Error(),
			}, err
		}

		return &tpvmcmm.VMResult{
			Code: tpvmcmm.ReturnCode_Ok,
		}, nil
	}

	if ncMethod.resultErrorOutIndex == 1 {
		if err, ok := callRtn[1].Interface().(error); ok && err != nil {
			return &tpvmcmm.VMResult{
				Code:   tpvmcmm.ReturnCode_MethodErr,
				ErrMsg: err.Error(),
			}, err
		}

		vmResult := &tpvmcmm.VMResult{
			Code: tpvmcmm.ReturnCode_Ok,
			Data: callRtn[0].Interface(),
		}

		eventhub.GetEventHubManager().GetEventHub(nodeID).Trig(ctx, eventhub.EventName_ContractInvoked, vmResultType)

		return vmResult, nil
	}

	err := fmt.Errorf("Execute unknown err for contract %s method %s", contractName, methodName)
	return &tpvmcmm.VMResult{
		Code:   tpvmcmm.ReturnCode_UnknownErr,
		ErrMsg: err.Error(),
	}, err
}

func (nvm *NativeVM) ExecuteContract(ctx *tpvmcmm.VMContext) (*tpvmcmm.VMResult, error) {
	if !nvm.Enable() {
		err := errors.New("Native vm not enable ")
		return &tpvmcmm.VMResult{
			Code:   tpvmcmm.ReturnCode_NotEnable,
			ErrMsg: err.Error(),
		}, err
	}

	if ctx.ContractName == "" || ctx.Method == "" {
		err := fmt.Errorf("Invalid vm ctx: contract %s method %s", ctx.ContractName, ctx.Method)
		return &tpvmcmm.VMResult{
			Code:   tpvmcmm.ReturnCode_InvalidVMCtx,
			ErrMsg: err.Error(),
		}, err
	}

	contractMap, ok := nvm.contractMethotds[ctx.ContractName]
	if !ok {
		err := fmt.Errorf("Can't find contract %s", ctx.ContractName)
		return &tpvmcmm.VMResult{
			Code:   tpvmcmm.ReturnCode_ContractNotFound,
			ErrMsg: err.Error(),
		}, err
	}

	ncMethod, ok := contractMap[ctx.Method]
	if !ok {
		err := fmt.Errorf("Can't find contract %s method %s", ctx.ContractName, ctx.Method)
		return &tpvmcmm.VMResult{
			Code:   tpvmcmm.ReturnCode_MethodNotFound,
			ErrMsg: err.Error(),
		}, err
	}

	if len(ctx.Args) != len(ncMethod.paramTypes)-1 {
		err := fmt.Errorf("Invalid parameters for contract %s method %s: expect len %d, actual %d", ctx.ContractName, ctx.Method, len(ncMethod.paramTypes)-1, len(ctx.Args))
		return &tpvmcmm.VMResult{
			Code:   tpvmcmm.ReturnCode_InvalidParam,
			ErrMsg: err.Error(),
		}, err
	}

	for i, paramType := range ncMethod.paramTypes[2:] {
		if ctx.Args[i].Type().Elem() != paramType {
			err := fmt.Errorf("Invalid parameters for contract %s method %s: parameter %d exected type %s", ctx.ContractName, ctx.Method, i, paramType.Name())
			return &tpvmcmm.VMResult{
				Code:   tpvmcmm.ReturnCode_InvalidParam,
				ErrMsg: err.Error(),
			}, err
		}
	}

	exeCtx := context.WithValue(ctx.Context, "VMServant", ctx.VMServant)

	var paramIns []reflect.Value
	paramIns = append(paramIns, ncMethod.receiver)
	paramIns = append(paramIns, reflect.ValueOf(exeCtx))
	paramIns = append(paramIns, ctx.Args...)

	return nvm.doExecute(exeCtx, ctx.NodeID, ctx.ContractName, ctx.Method, ncMethod, paramIns)
}